package report

import (
	"sort"

	"github.com/victorpero/amex-grocery-splitter-se/internal/matcher"
	"github.com/victorpero/amex-grocery-splitter-se/internal/split"
	"github.com/victorpero/amex-grocery-splitter-se/internal/transaction"
)

type Analysis struct {
	Matched   []transaction.Transaction
	Unmatched []transaction.Transaction
	Result    split.Result
}

func Analyze(transactions []transaction.Transaction, groceryMatcher *matcher.PrefixMatcher, amountMode split.AmountMode) Analysis {
	matched := make([]transaction.Transaction, 0)
	unmatched := make([]transaction.Transaction, 0)

	for _, tx := range transactions {
		if groceryMatcher.IsGrocery(tx.Description) {
			matched = append(matched, tx)
		} else {
			unmatched = append(unmatched, tx)
		}
	}

	SortTransactions(matched)
	SortTransactions(unmatched)

	return Analysis{
		Matched:   matched,
		Unmatched: unmatched,
		Result:    split.Calculate(matched, amountMode),
	}
}

func SortTransactions(transactions []transaction.Transaction) {
	sort.SliceStable(transactions, func(i, j int) bool {
		if transactions[i].Date.Equal(transactions[j].Date) {
			return transactions[i].Description < transactions[j].Description
		}
		return transactions[i].Date.Before(transactions[j].Date)
	})
}

func DisplayAmountCents(amount int64, mode split.AmountMode) int64 {
	if mode == split.AmountModeAbsolute && amount < 0 {
		return -amount
	}
	return amount
}
