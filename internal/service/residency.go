package service

import (
	"github.com/evgenii-ev/go-tax-bot/internal/model"
)

// Storage is the minimal persistence interface the service depends on.
type Storage interface {
	Load(int64) (model.UserData, error)
	Save(model.UserData) error
}

type Calculator struct {
	st Storage
}

func NewCalculator(st Storage) *Calculator {
	return &Calculator{st: st}
}

// --- Persistence helpers --------------------------------------------------

func (c *Calculator) LoadUser(chatID int64) (model.UserData, error) {
	return c.st.Load(chatID)
}

func (c *Calculator) SaveUser(data model.UserData) error {
	return c.st.Save(data)
}

// --- Business logic -------------------------------------------------------

// DaysPerCountry returns map[ISO2]days for the supplied periods.
func (c *Calculator) DaysPerCountry(periods []model.Period) map[string]int {
	res := make(map[string]int)

	for _, p := range periods {
		start := p.In
		if start == nil {
			// Special sem‑antics: lived in that country before Out forever
			s := p.Out
			start = &s
		}
		for d := *start; !d.After(p.Out); d = d.AddDate(0, 0, 1) {
			res[iso2(p.Country)]++
		}
	}

	return res
}

// Helper: human friendly country → ISO‑2 code.  In real app load a table.
func iso2(country string) string {
	switch country {
	case "Россия", "Russia":
		return "RU"
	case "Грузия", "Georgia":
		return "GE"
	default:
		return country // already ISO2?
	}
}
