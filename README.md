# Golang AWS Lambda function

The Go test lambda was set up to test out Go in an AWS Lambda + API Gateway to
see how they work together. There have been several iterations:

1. Lambda + API GW with no Auth
2. Lambda + API GW with Cognito pools
3. Lambda + API GW with Lambda Authorizer auth

## 1. Lambda + API GW with Cognito pools

TO invoke the Lambda directly, use the following bin script

```sh
./bin/invoke.sh
```

## 2. Lambda + API GW with Cognito pools

Cognito was an interesting experiment, but I don't think it's the Auth solution
I'm looking for. It's possible to set up Cognito Identity pools to allow AWS
access via issuing temporary tokens. While this could work, it would have a
very AWS-specific interface. The client (javascript) would need to authenticate
to AWS Cognito and retrieve a Token - then pass that token along to API
Gateway. While this seems possible, it doesn't work well if you want to use a
different identity provider as the main source of truth.

In my case, I want to use Keycloak as the source of truth for which users have
which permissions. While it would be possible to login via Keycloak and grant
access tokens to the Frontend for AWS resources, this feels like a
Cloud-specific solution - effectively tying my front end to AWS. Instead, I'd
prefer to send my OIDC identity token and use that as a mechanism for
authentication.

I set up a Cognito User pool, User group and Identity pool manually, and was
able to navigate to a Cognito-generated auth endpoint to retrieve an access
token.

My Cognito login
https://test-idp-andrewslai.auth.us-east-1.amazoncognito.com/login?response_type=code&client_id=3ro047v8l2lrrpctl6nl68vlt&redirect_uri=https://andrewslai.com

## 3. Lambda + API GW with Lambda Authorizer auth

The Lambda + API GW solution seems to fit my needs. I want to be able to
authenticate my users using Keycloak, and have my API Gateway check if the
user's group/identity allows them access to specific resources. This seems to fit well

https://repost.aws/knowledge-center/decode-verify-cognito-json-token
https://medium.com/@chaim_sanders/validating-okta-access-tokens-in-python-with-pyjwt-33b5a66f1341
https://gist.github.com/bendog/44f21a921f3e4282c631a96051718619

To invoke API GW with the Lambda authorizer auth

With a valid token, this should work

```sh
curl -v -H 'Authorization: Bearer YOUR-TOKEN-HERE' https://c10qtm1qd8.execute-api.us-east-1.amazonaws.com/prod
```

Unauthorized examples

```sh
curl -v https://c10qtm1qd8.execute-api.us-east-1.amazonaws.com/prod
curl -v -H 'Authorization: Bearer x' https://c10qtm1qd8.execute-api.us-east-1.amazonaws.com/prod
curl -v -H 'Bearer x' https://c10qtm1qd8.execute-api.us-east-1.amazonaws.com/prod
```

TODO: Add jwt library in lambda layer
TODO: Add group/role to JWT
https://stackoverflow.com/questions/56362197/keycloak-oidc-retrieve-user-groups-attributes

###

## Resources:

https://www.alexedwards.net/blog/serverless-api-with-go-and-aws-lambda
https://www.alexedwards.net/blog/which-go-router-should-i-use

For structured loggning log/slog, but must be on Go 1.21.0 or later
https://pkg.go.dev/log/slog?tab=versions

There are two types of API GW resources - REST APIs (v1) and HTTP APIs (v2). We
cannot mix the MUX/routers, because each API GW expects results in a slightly
different format (specifically, I believe V2 expectes a "cookies" field that is
not present in V1)

## Testing

```go

// setFieldValue is only for testing
func setFieldValue(target any, fieldName string, value any) {
	rv := reflect.ValueOf(target)
	for rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if !rv.CanAddr() {
		panic("target must be addressable")
	}
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf(
			"unable to set the '%s' field value of the type %T, target must be a struct",
			fieldName,
			target,
		))
	}
	rf := rv.FieldByName(fieldName)

	reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

type TestResponseRecorder struct {
	Code      int
	HeaderMap http.Header
	Body      *bytes.Buffer
	Flushed   bool

	Result      *http.Response // cache of Result's return value
	SnapHeader  http.Header    // snapshot of HeaderMap at first Write
	WroteHeader bool
}

func (trr TestResponseRecorder) ToResponseRecorder() *httptest.ResponseRecorder {
	rr := &httptest.ResponseRecorder{
		Code:      trr.Code,
		HeaderMap: trr.HeaderMap,
		Body:      trr.Body,
		Flushed:   trr.Flushed,
	}
	if trr.Result != nil {
		setFieldValue(rr, "result", trr.Result)
	}
	if trr.SnapHeader != nil {
		setFieldValue(rr, "snapHeader", trr.SnapHeader)
	}
	setFieldValue(rr, "wroteHeader", trr.WroteHeader)
	return rr
}
```

## Database

On AWS, I created a new database inside my RDS instance - `go_websocket`. This was to share resources, yet still allow my lambda to operate in a clean DB environment

First, on AWS, I created a new user for the database and granted the role to my superuser (to allow the superuser to modify the role). Then, I created a new database.

```sql
CREATE USER go_websocket_user with encrypted password '';
GRANT go_websocket_user to andrewslai;
CREATE DATABASE go_websocket;
GRANT ALL PRIVILEGES ON DATABASE go_websocket to go_websocket_user;
```

## Hot reloading

https://github.com/cespare/reflex

```sh
reflex -r '\.go' -s go run pkg/main.go
```

## Testing APIs

```sh
eval `./bin/get-token.sh`
curl -H "Authorization: Bearer $KEYCLOAK_ACCESS_TOKEN" https://c10qtm1qd8.execute-api.us-east-1.amazonaws.com/prod/hello
```

## Networking

A Lambda function inside a VPC needs to be on a private subnet.
Access to the outside internet happens by means of an NAT gateway.

Unfortunately, AWS services are exposed on the public internet (Secretsmanager
etc). So to get that access, you either need to set up the NAT gateway ($42/month), or
create a VPC endpoint ($7/month)

https://nodogmablog.bryanhogan.net/2022/06/accessing-the-internet-from-vpc-connected-lambda-functions-using-a-nat-gateway/
https://www.techtarget.com/searchcloudcomputing/answer/How-do-I-configure-AWS-Lambda-functions-in-a-VPC
https://www.alexdebrie.com/posts/aws-lambda-vpc/
https://aws.amazon.com/blogs/architecture/overview-of-data-transfer-costs-for-common-architectures/

Seems like the VPC endpoint worked - I still don't have internet access on the
Lambda, but it's better than putting the Lambda on the VPC and setting it up
with a NAT.

NEXT: try lambda without VPC

## Hot reloading

I am using [Air](https://github.com/cosmtrek/air) for hot reloading. The configuration is in `.air.toml`.

To start the service with hot reloading:

```sh
air
```

## Migrations

```sh
goose create <<NAME-HERE>> sql
```

goose create create-photo-metadata-table sql
goose -dir migrations postgres "user=$DB_USER dbname=$DB_NAME" reset

## Usage

Local connections

```sh
air
./bin/connect
```

Connecting in prod

```sh
./bin/connect-api-gw.sh
```

### Actions

```json
{ "action": "broadcast", "message": "Hello" }
```

```json
{ "action": "whoall" }
```

# Resources

https://github.com/misikdmytro/go-ws-api-gateway/blob/master/internal/handler/connect.go
https://docs.aws.amazon.com/apigateway/latest/developerguide/websocket-api-chat-app.html

## TODO

1: Expand the handler to handler $connect messages
2: Put a Keycloak Authorizer in front of `$connect`
