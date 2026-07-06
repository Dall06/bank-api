package logs

// BlockedFields is a list of field names whose values should be masked in logs.
var BlockedFields = []string{
	"password", "password_hash", "pass", "pwd",
	"token", "access_token", "refresh_token", "id_token",
	"api_key", "apikey", "x_api_key", "x-api-key", "secret", "secret_key", "secretkey",
	"authorization", "auth", "bearer",
	"cookie", "set-cookie",
	"ssn", "nif", "dni",
	"credit_card", "card_number", "pan", "cvv", "cvv2",
	"account_number", "routing_number", "iban", "sort_code", "clabe", "account_holder",
	"birth_date", "dob",
	"private_key", "privatekey", "rsa_private",
	"otp", "mfa", "totp",
	"email", "phone", "address",
}
