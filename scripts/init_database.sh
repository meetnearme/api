#!/bin/bash

export AWS_SECRET_ACCESS_KEY="test"
export AWS_ACCESS_KEY_ID="test"
export AWS_REGION="us-east-1"

echo "start sleep"
sleep 5
echo "end sleep"

aws dynamodb create-table --cli-input-json file://internal/database/db_seeds/create_event_table.json --endpoint-url http://dynamodb-local:8000 --region us-east-1
aws dynamodb list-tables --endpoint-url http://dynamodb-local:8000 --region us-east-1

aws dynamodb batch-write-item --request-items file://internal/database/db_seeds/seed_user_records.json --endpoint-url http://dynamodb-local:8000 --region us-east-1

echo "database seed complete"
if [ $1 == "--forever" ]
then
    echo "staying up to keep dependent services happy"
    sleep 10000
fi
