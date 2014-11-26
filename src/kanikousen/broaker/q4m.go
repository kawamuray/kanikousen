package broaker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strings"

	k "../"
)

type Q4mBroaker struct {
	boat *k.Boat
	conn *sql.DB
}

type Q4mJobTxn struct {
	*sql.Tx
}

func init() {
	k.RegisterBroaker("q4m", Q4mCreateFromConfig)
}

func Q4mCreateFromConfig(boat *k.Boat, cfgs map[string]string) (k.Broaker, error) {
	dsn, ok := cfgs["dsn"]
	if !ok {
		return nil, fmt.Errorf("Error: Q4M driver requires 'dsn' config specified")
	}

	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(int(boat.MaxWorkers + 1))

	// We have to check if the db configuration is valid to at least
	// making a connection and also the DBMS is working fine.
	err = conn.Ping()
	if err != nil {
		return nil, fmt.Errorf("Unable to confirm connection for database: %s", err)
	}

	brk := &Q4mBroaker{
		boat: boat,
		conn: conn,
	}
	return brk, nil
}

func (txn *Q4mJobTxn) rowsCount() (uint, error) {
	sql := `SELECT COUNT(*) FROM kanikousen_job WHERE status = 'WAIT'`
	var count uint
	err := txn.QueryRow(sql).Scan(&count)
	return count, err
}

func (txn *Q4mJobTxn) fetchOnce(chanJob chan *k.Job, sched *k.Schedule, queue string) {
	pushBack := true
	defer func() {
		if pushBack {
			txn.queueMarkFinish(true)
			txn.Commit()
		}
	}()

	// Check number of jobs satisfaction if necessary
	if sched.JobsCount != 0 {
		cnt, err := txn.rowsCount()
		if err != nil {
			log.Println("ERROR: failed to get rows count: ", err)
			return
		}
		if cnt < sched.JobsCount {
			return // This is not my turn
		}
	}

	var (
		id       uint
		taskId   string
		argsData []byte
	)
	err := txn.QueryRow(`SELECT * FROM `+queue).Scan(&id, &taskId, &argsData)
	if err != nil {
		log.Println("ERROR: select queue: ", err)
		return
	}
	log.Println("job selected id=", id)
	var args []string
	err = json.Unmarshal(argsData, &args)
	if err != nil {
		log.Println("malformed JSON for column args: ", err)
		_, err = txn.Exec(`UPDATE kanikousen_job SET status = 'FAILED' WHERE id = ?`, id)
		if err != nil {
			log.Println("ERROR: updating kanikousen_job table: ", err)
		}
		return
	}

	chanJob <- &k.Job{
		Id:     id,
		TaskId: taskId,
		Args:   args,
		Data:   txn,
	}
	pushBack = false
}

func buildSqlSelectQueuesRandom() string {
	// TODO bad performance algorithm
	return `SELECT name FROM kanikousen_queues ORDER BY RAND()`
}

func buildSqlSelectQueues(queues []string) (string, []interface{}) {
	if len(queues) == 0 || queues[0] == "*" {
		return buildSqlSelectQueuesRandom(), nil
	}

	// At this point, `queues` contains at least an item and the first element isn't a wildcard
	sql := `SELECT name FROM (%s) AS a WHERE a.name IN (SELECT name FROM kanikousen_queues)`
	literals := ""
	binds := make([]interface{}, len(queues))
	for i, q := range queues {
		if q == "*" { // Wildcard! more iteration is meaningless.
			binds = binds[0:i]
			sql += " UNION DISTINCT " + buildSqlSelectQueuesRandom()
			break
		} else {
			literals += "SELECT ? AS name UNION DISTINCT "
			binds[i] = q
		}
	}
	// Workaround for this bug: http://bugs.mysql.com/bug.php?id=71577
	literals += "SELECT NULL"

	return fmt.Sprintf(sql, literals), binds
}

func (txn *Q4mJobTxn) fetchQueueCandidates(sql string, binds []interface{}) ([]interface{}, error) {
	rows, err := txn.Query(sql, binds...)
	if err != nil {
		log.Println("ERROR: begin transaction: ", err)
		return nil, err
	}
	var queues []interface{}
	for rows.Next() {
		var q string
		rows.Scan(&q)
		queues = append(queues, q)
	}
	return queues, nil
}

func (txn *Q4mJobTxn) selectQueue(queues []interface{}) (int, error) {
	waitSql := fmt.Sprintf(`SELECT queue_wait(%s, 60)`,
		strings.Join(strings.Split(strings.Repeat("?", len(queues)), ""), ","))
	var qi int
	err := txn.QueryRow(waitSql, queues...).Scan(&qi)
	if err != nil {
		log.Println("ERROR: waiting queue: ", err)
		return -1, err
	}
	return qi, nil
}

func (q4m *Q4mBroaker) FetchLoop(chanJob chan *k.Job, sched *k.Schedule) {
	sql, binds := buildSqlSelectQueues(sched.Queues)
	log.Println(sql, binds)

	for {
		tx, err := q4m.conn.Begin()
		if err != nil {
			log.Println("ERROR: begin transaction: ", err)
			continue
		}
		txn := &Q4mJobTxn{tx}

		queues, err := txn.fetchQueueCandidates(sql, binds)
		if err != nil {
			txn.Commit()
			continue
		}

		qi, err := txn.selectQueue(queues)
		if err != nil {
			txn.Commit()
			continue
		}
		if qi > 0 {
			// Wrap it!
			txn.fetchOnce(chanJob, sched, queues[qi-1].(string))
		} else {
			txn.Commit()
		}
	}
}

func (txn *Q4mJobTxn) queueMarkFinish(abort bool) error {
	var fn string
	if abort {
		fn = `queue_abort()`
	} else {
		fn = `queue_end()`
	}
	_, err := txn.Exec(`SELECT ` + fn)
	if err != nil {
		log.Printf("ERROR: executing %s: %s\n", fn, err)
	}
	return err
}

func (txn *Q4mJobTxn) updateJobStatus(jobId uint, status string) error {
	_, err := txn.Exec(`UPDATE kanikousen_job SET status = ? WHERE id = ?`,
		status, jobId)
	if err != nil {
		log.Printf("ERROR: updating kanikousen_job table: ", err)
	}
	return err
}

func (q4m *Q4mBroaker) Failed(job *k.Job, err error) {
	txn := job.Data.(*Q4mJobTxn)
	log.Printf("job %d failed: task=%s, args=%s err=%s\n",
		job.Id, job.TaskId, job.Args, err)
	// This error can't be recovered by retrying
	txn.queueMarkFinish(false)
	txn.updateJobStatus(job.Id, "FAILED")
	err = txn.Commit()
	if err != nil {
		log.Printf("ERROR: finishing transaction: ", err)
	}
}

func (q4m *Q4mBroaker) handleFailedJob(job *k.Job) {
	txn := job.Data.(*Q4mJobTxn)
	var failCount uint
	// Remember, we're in the transaction ;)
	err := txn.QueryRow(`SELECT fail_count FROM kanikousen_job
                         WHERE id = ?`, job.Id).Scan(&failCount)
	if err != nil {
		log.Printf("ERROR: can't fetch fail_count for job %d: %s\n", job.Id, err)
	} else {
		if failCount >= q4m.boat.MaxFailCount {
			err := txn.updateJobStatus(job.Id, "FAILED")
			if err != nil {
				log.Printf("ERROR: can't update status for job %d: %s\n", job.Id, err)
			} else {
				// Sorry, you don't have more chance.
				txn.queueMarkFinish(false)
				return
			}
		} else {
			_, err := txn.Exec(`UPDATE kanikousen_job SET fail_count = ?
                                WHERE id = ?`, failCount+1, job.Id)
			if err != nil {
				log.Printf("ERROR: can't update fail_count for job %d: %s\n", job.Id, err)
			}
		}
	}
	txn.queueMarkFinish(true)
}

func (q4m *Q4mBroaker) Finish(job *k.Job, status int) {
	txn := job.Data.(*Q4mJobTxn)
	log.Printf("job %d finished: task=%s, args=%s status=%d\n",
		job.Id, job.TaskId, job.Args, status)

	if status == 0 {
		txn.queueMarkFinish(false)
		txn.updateJobStatus(job.Id, "SUCCEED")
	} else {
		q4m.handleFailedJob(job)
	}
	err := txn.Commit()
	if err != nil {
		log.Printf("ERROR: finishing transaction: ", err)
	}
}

func (q4m *Q4mBroaker) Close() {
	q4m.conn.Close()
}
