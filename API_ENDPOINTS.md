## Users

1. Create User

```bash

# example of all fields
curl -X POST https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/users \
 -H "Content-Type: application/json" \
 -d '{
     "name": "B Doe",
     "email": "b.doe@example.com",
     "role": "standard_user",
     "address": "123 Main St, New York 10001 USA",
     "phone": "+123456789",
     "profilePictureUrl": "https://example.com/profile.jpg"
 }'

 # only required fields
 curl -X POST  https://qnu7q7ch56.execute-api.us-east-1.amazonaws.com/api/users \
 -H "Content-Type: application/json" \
 -d '{
     "name": "John Doe",
     "email": "john.doe@example.com",
     "role": "standard_user"
 }'

 
```

2. Get User by ID
```bash
curl -X GET <instance_url>/users/<:id>

curl -X GET https://qnu7q7ch56.execute-api.us-east-1.amazonaws.com/api/users/5648e67b-2e00-4f2d-8498-651382a6ddee

```

3. Get Users
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/users 
```

3. Update user
```bash
curl -X PUT https://qnu7q7ch56.execute-api.us-east-1.amazonaws.com/api/users/bb165e53-31c8-4085-9382-578ac7df6812 \
-H "Content-Type: application/json" \
-d '{
    "name": "New name",
    "email": "new.name@example.com",
    "address_street": "51 main street",
    "address_city": "New Haven",
    "address_zip_code": "51515",
    "address_country": "USA",
    "phone": "+1234567890",
    "profile_picture_url": "http://example.com/profile.jpg",
    "role": "organization_user"
}'

```

4. Delete User
```bash
curl -X DELETE https://qnu7q7ch56.execute-api.us-east-1.amazonaws.com/api/users/bb165e53-31c8-4085-9382-578ac7df6812 \
```

##Transactions

1. Create Transaction
```bash
curl -X POST https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/transactions \
-H "Content-Type: application/json" \
-d '{
  "userId": "ea49a5f8-e27c-47b0-8237-6f6f380a048c",
  "amount": "100.00",
  "currency": "USD",
  "transaction_type": "credit",
  "status": "completed",
  "description": "Car repair"
}'

```

2. Update Transaction

<!-- must use new user id -->
```bash
curl -X PUT https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/transactions \
-H "Content-Type: application/json" \
-d '{
  "userId": "5648e67b-2e00-4f2d-8498-651382a6ddee",
  "amount": "150.00",
  "currency": "USD",
  "transaction_type": "debit",
  "status": "failed",
  "description": "Payment for services refunded"
}'
```
3. Delete transaction
```bash
curl -X DELETE https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/transactions/cbe40f6b-10c6-4e8c-a54d-01d2cfde16d7
```

4. Get Transaction by ID
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/transactions/cbe40f6b-10c6-4e8c-a54d-01d2cfde16d7
```

5. Get Transactions by UserID
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/transactions/user/ea49a5f8-e27c-47b0-8237-6f6f380a048c
```

## Purchasables

1. Create Purchasable
```bash
curl -X POST https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/purchasables \
     -H "Content-Type: application/json" \
     -d '{
           "user_id": "ea49a5f8-e27c-47b0-8237-6f6f380a048c",
           "name": "Sample Item 2",
           "item_type": "ticket",
           "cost": 69.99,
           "currency": "USD",
           "donation_ratio": 0.10,
           "inventory": 100,
           "charge_recurrence_interval": "month",
           "charge_recurrence_interval_count": 1,
           "charge_recurrence_end_date": "2024-12-31T23:59:59Z"
         }'

```

2. Update Purchasable
```bash
curl -X PUT https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/purchasables/47f5e966-2190-4ae0-9344-b8aa80202e8a \
     -H "Content-Type: application/json" \
     -d '{
           "name": "Updated Item",
           "item_type": "membership",
           "cost": 99.99,
           "currency": "USD",
           "donation_ratio": 0.15,
           "inventory": 200,
           "charge_recurrence_interval": "year",
           "charge_recurrence_interval_count": 1,
           "charge_recurrence_end_date": "2025-12-31T23:59:59Z"
         }'

```
3. Get Purchasable by ID
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/purchasables/47f5e966-2190-4ae0-9344-b8aa80202e8a \
     -H "Content-Type: application/json"

```
4. Get Purchasables by UserID
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/purchasables/user/ea49a5f8-e27c-47b0-8237-6f6f380a048c\
     -H "Content-Type: application/json"
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/purchasables/user/40164b69-b2fe-4706-aa12-7d6ec3845c95 \
     -H "Content-Type: application/json"
```
5. Delete Purchasable
```bash
curl -X DELETE https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/purchasables/47f5e966-2190-4ae0-9344-b8aa80202e8a \
     -H "Content-Type: application/json"

```


## Event RSVPs
1. Create EventRsvp
```bash
curl -X POST https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/event-rsvps \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "ea49a5f8-e27c-47b0-8237-6f6f380a048c",
    "event_id": "6ce1be30-f700-475c-b84a-49af0c73f337",
    "event_source_type": "internalRecurrence",
    "status": "Yes"
}'
```

2. GET EventRsvp By ID
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/event-rsvps/32bf7a56-3267-4456-a325-eb90adbc2935 \
  -H "Content-Type: application/json"

```

3. GET EventRsvp By UserID
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/event-rsvps/user/ea49a5f8-e27c-47b0-8237-6f6f380a048c \
  -H "Content-Type: application/json"

```
4. GET EventRsvp By EventID
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/event-rsvps/event/6ce1be30-f700-475c-b84a-49af0c73f337 \
  -H "Content-Type: application/json"
```

5. Update EventRsvp
```bash
curl -X PUT https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/event-rsvps/32bf7a56-3267-4456-a325-eb90adbc2935 \
  -H "Content-Type: application/json" \
  -d '{
    "status": "Maybe"
}'

```

6. Delete EventRsvp
```bash
curl -X DELETE https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/event-rsvps/32bf7a56-3267-4456-a325-eb90adbc2935 \
  -H "Content-Type: application/json"

```

## Registration Fields
1. Create Registration Fields
```bash
curl -X POST https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/registration-fields \
-H "Content-Type: application/json" \
-d '{
  "name": "Gaming Night",
  "type": "game event",
  "options": "",
  "required": true,
  "default": "",
  "placeholder": "Enter your email",
  "description": "User email for registration"
}'

```
2. Get Registration Fields
```bash
curl -X GET https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/registration-fields/c0e7fcdc-8ca6-4018-9f72-e8394b566c1f  \
-H "Content-Type: application/json"

```
3. Update Registration Fields
```bash
curl -X PUT https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/registration-fields/c0e7fcdc-8ca6-4018-9f72-e8394b566c1f \
-H "Content-Type: application/json" \
-d '{
  "name": "username",
  "type": "text",
  "options": "here are some cool options",
  "required": false,
  "default": "this is now default",
  "placeholder": "Enter your username",
  "description": "User username for registration"
}'
```
4. Delette Registration Fields
```bash
curl -X DELETE https://wxw9ojvrcl.execute-api.us-east-1.amazonaws.com/api/registration-fields/c0e7fcdc-8ca6-4018-9f72-e8394b566c1f \
-H "Content-Type: application/json"
```
