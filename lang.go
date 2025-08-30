package main

// Language codes identify which language's error strings to use.
const (
	LANG_SPANISH = "es"
	LANG_ENGLISH = "en"
)

// Error codes represent specific error conditions.
const (
	ERR_ACCOUNT_NUMBER_INVALID = iota
	ERR_ACCOUNT_NOT_FOUND
	ERR_INCORRECT_PASSWORD
	ERR_TRANSFER_AMOUNT_INVALID
	ERR_TRANSFER_DATE_INVALID
	ERR_NEW_PASSWORDS_MISMATCH
	ERR_CURRENT_PASSWORD_INCORRECT
	ERR_TRANSACTION_ID_INVALID
	ERR_LETTER_ID_INVALID
)

const (
	ERR_DOC_NOT_FOUND = "not found"
	ERR_LETTER_NOT_IN_INBOX = "not in inbox"
	ERR_TRANSFER_TO_SELF = "transfer to self"
	ERR_INSUFFICIENT_FUNDS = "no funds"
	ERR_NEGATIVE_TRANSFER_AMOUNT = "negative money"
	ERR_TIME_TRAVEL_IMPOSSIBLE = "time travel"
	ERR_RECIPIENT_ACCOUNT_NOT_FOUND = "no recipient"
	ERR_REVOKE_NOT_ALLOWED = "no revoke"
)

// SpanishErrors holds the Spanish translations for the error codes.
var SpanishErrors = []string{
	"El número de cuenta debe ser un entero positivo",
	"Cuenta no encontrada (debe darse de alta)",
	"Introduzca sus datos correctamente.",
	"El importe a transferir debe ser un entero positivo",
	"La fecha debe ser un entero positivo",
	"Las nuevas contraseñas no coinciden",
	"La contraseña actual no es correcta",
	"Identificador de transacción erróneo",
	"Identificador de carta erróneo",
}

// EnglishErrors holds the English translations for the error codes.
var EnglishErrors = []string{
	"The account number must be a positive integer",
	"Account not found (you must register)",
	"Please enter your details correctly.",
	"The amount to transfer must be a positive integer",
	"The date must be a positive integer",
	"The new passwords do not match",
	"The current password is not correct",
	"Incorrect transaction identifier",
	"Incorrect letter identifier",
}

var ErrorStrings = map[string][]string {
	LANG_ENGLISH: EnglishErrors,
	LANG_SPANISH: SpanishErrors,
}

var BackendErrors = map[string]map[string]string {
	LANG_ENGLISH: {
		ERR_DOC_NOT_FOUND : "Document not found",
		ERR_LETTER_NOT_IN_INBOX : 	"Letter not found in your inbox",
		ERR_TRANSFER_TO_SELF : 	"You cannot transfer money to your own account",
		ERR_INSUFFICIENT_FUNDS : 	"You do not have sufficient funds",
		ERR_NEGATIVE_TRANSFER_AMOUNT : 	"It is not possible to transfer a negative amount",
		ERR_TIME_TRAVEL_IMPOSSIBLE : 	"Time travel is not possible...",
		ERR_RECIPIENT_ACCOUNT_NOT_FOUND : 	"The account you are trying to transfer to does not exist",
		ERR_REVOKE_NOT_ALLOWED : 	"The transaction does not meet the requirements to be revoked by you. Please contact the Bank to resolve the issue.",
	},
	LANG_SPANISH: {
		ERR_DOC_NOT_FOUND : "No se encontró el documento", 
		ERR_LETTER_NOT_IN_INBOX : 	"No se encontró la carta en su buzón", 	
		ERR_TRANSFER_TO_SELF : 	"No se puede transferir dinero a su misma cuenta", 	
		ERR_INSUFFICIENT_FUNDS : 	"No dispone de los fondos suficientes", 	
		ERR_NEGATIVE_TRANSFER_AMOUNT : 	"No es posible transferir un importe negativo", 	
		ERR_TIME_TRAVEL_IMPOSSIBLE : 	"No es posible viajar en el tiempo...", 	
		ERR_RECIPIENT_ACCOUNT_NOT_FOUND : 	"La cuenta a la que está intentando ordernar la transferencia no existe", 	
		ERR_REVOKE_NOT_ALLOWED : 	"La transacción no cumple los requerimientos para ser revocada por usted. Contacte con el Banco para resolver el problema.", 
	},
}

func GetBackendError(lang string, e string) string {
	s, ok := BackendErrors[lang][e]
	if ok {
		return s
	} else {
		return e
	}
}