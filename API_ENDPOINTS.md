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
