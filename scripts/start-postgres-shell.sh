#!/bin/bash

docker exec -it $(docker ps --filter name=unpaper_pg_1 --format "{{.Names}}") /bin/bash