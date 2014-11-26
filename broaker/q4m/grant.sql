-- This SQL must be run within root user connected from local unix socket
-- Following line would solve the problem:
-- nsenter --target $(docker inspect --format '{{ .State.Pid }}' kanikousen-q4m) --mount mysql -uroot < grant.sql

GRANT SELECT ON TABLE kanikousen_job TO 'foo'@'%';
GRANT CREATE, SELECT, INSERT ON TABLE * TO 'foo'@'%';
GRANT DELETE ON TABLE kanikousen_queues TO 'foo'@'%';
