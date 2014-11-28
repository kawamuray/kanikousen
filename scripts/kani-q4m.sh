#!/bin/bash
# !!CAUTION!! This scripts isn't safe to use for untrusted input
set -e

Q4M_HOST=""               # Default value 'H'
Q4M_USER=""               # Default value 'u'
Q4M_QUEUE="default_queue" # Default value 't'
Q4M_DB="kanikousen"

MYSQL=$(which mysql)

usage() {
    cat <<EOF
Usage: $0 ACTION [options] [arguments]

Actions:
  create_queue QUEUE_NAME            -- Create a queue named QUEUE_NAME
  remove_queue QUEUE_NAME            -- Remove a queue named QUEUE_NAME
  insert_job TASK_ID [ARG1, ARG2...] -- Insert a job that has TASK_ID with arguments
    -t QUEUE_NAME  -- Specify a queue name to insert into(default: default_queue)
  poll_job JOB_ID                    -- Poll job until it exits

Common options:
  -h          -- Show this help
  -H HOST     -- Specify a host of Q4M server
  -u USER     -- Specify a user to connect Q4M server

Example usage:
  kani-q4m.sh create_queue foo_queue
  id1=\$(kani-q4m.sh insert_job -t foo_queue foo/bar_task fileA.txt)
  id2=\$(kani-q4m.sh insert_job -t foo_queue foo/bar_task fileB.txt)
  kani-q4m.sh poll_job \$id1
  kani-q4m.sh poll_job \$id2
  kani-q4m.sh remove_queue foo_queue
EOF
}

mysql_() {
    $MYSQL -u$Q4M_USER -h$Q4M_HOST $* $Q4M_DB
}

checkifnull() {
    if [ -z "$2" ]; then
        echo "$1 is missing" >&2
        usage >&2
        exit 1
    fi
}

create_queue() {
    NAME="$1"
    checkifnull NAME $NAME
    mysql_ <<EOF
SET AUTOCOMMIT = 0;
BEGIN;
CREATE TABLE $NAME LIKE default_queue;
INSERT kanikousen_queues SET name='$NAME';
COMMIT;
EOF
}

remove_queue() {
    NAME="$1"
    checkifnull NAME $NAME
    mysql_ <<EOF
DELETE FROM kanikousen_queues WHERE name='$NAME';
-- RENAME TABLE $NAME TO __deleted__$NAME;
EOF
}

insert_job() {
    TASK_ID="$1"
    checkifnull TASK_ID $TASK_ID; shift
    ARGS=$(python -c 'import json, sys; json.dump(sys.argv[1:], sys.stdout)' $*)
    mysql_ -N <<EOF
SET AUTOCOMMIT = 0;
BEGIN;
INSERT kanikousen_job SET task_id = '$TASK_ID', args = '$ARGS';
SET @job_id=(SELECT LAST_INSERT_ID());
INSERT $Q4M_QUEUE SET id = @job_id, task_id = '$TASK_ID', args = '$ARGS';
COMMIT;
SELECT @job_id;
EOF
}

poll_job() {
    JOB_ID="$1"
    checkifnull JOB_ID $JOB_ID
    while true; do
        status=$(mysql_ -N <<EOF
SELECT status FROM kanikousen_job WHERE id = $JOB_ID AND status != 'WAIT';
EOF
        )
        if [ -n "$status" ]; then
            break
        fi
        sleep 1
    done
    echo $status
}

ACTION="$1"
checkifnull ACTION $ACTION; shift
if ! LC_ALL=C LANG=C type $ACTION | grep "function$" >/dev/null; then
    echo "unkown action '$ACTION'" >&2
    usage >&2
    exit 1
fi

while getopts "hu:H:t:" opt; do
    case $opt in
        h)  usage
            exit 0
            ;;
        u) Q4M_USER=$OPTARG
            ;;
        H) Q4M_HOST=$OPTARG
            ;;
        t) Q4M_QUEUE=$OPTARG
            ;;
    esac
done
shift $((OPTIND - 1))

# Prereq checks
checkifnull Q4M_HOST $Q4M_HOST
checkifnull Q4M_USER $Q4M_USER
checkifnull Q4M_QUEUE $Q4M_QUEUE
checkifnull Q4M_DB $Q4M_DB

$ACTION $*
exit 0
