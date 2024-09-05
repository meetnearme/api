## Users

1. Create User

```bash

# example of all fields
curl -X POST  https://qnu7q7ch56.execute-api.us-east-1.amazonaws.com/api/users \
 -H "Content-Type: application/json" \
 -d '{
     "name": "B Doe",
     "email": "b.doe@example.com",
     "role": "standard_user",
     "addressStreet": "123 Main St",
     "addressCity": "New York",
     "addressZipCode": "10001",
     "addressCountry": "USA",
     "phone": "+123456789",
     "profilePictureUrl": "https://example.com/profile.jpg",
     "organizationUserId": ""
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
curl -X GET https://qnu7q7ch56.execute-api.us-east-1.amazonaws.com/api/users
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
