-- Table for stacking jobs
CREATE TABLE kanikousen_job (
  id         BIGINT UNSIGNED                   NOT NULL AUTO_INCREMENT,
  task_id    VARCHAR(255)                      NOT NULL,
  args       BLOB                              NOT NULL,
  status     ENUM('WAIT', 'SUCCEED', 'FAILED') NOT NULL DEFAULT 'WAIT',
  fail_count INT UNSIGNED                      NOT NULL DEFAULT 0,
  PRIMARY KEY(id),
  KEY(status),
  KEY(task_id, status)
) ENGINE=InnoDB;

-- Just a list of created queues
CREATE TABLE kanikousen_queues (
  id   INT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  PRIMARY KEY(id),
  UNIQUE(name)
) ENGINE=InnoDB;

-- Default Q4M queue
CREATE TABLE default_queue (
  id      INT UNSIGNED NOT NULL,
  task_id VARCHAR(255) NOT NULL,
  args    BLOB         NOT NULL
) ENGINE=queue;
INSERT kanikousen_queues SET name='default_queue';
