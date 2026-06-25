package transaction

import "time"

// Transaction is the normalized representation used by the rest of the app.
// AmountCents keeps the sign found in the CSV. Split calculations can choose
// whether to preserve signs or treat all matched purchases as positive costs.
type Transaction struct {
	Date        time.Time
	Description string
	AmountCents int64
	SourceFile  string
	SourceLine  int
}
