package emailverifier

import (
	"net"
	"sort"
)

type Mx struct {
	HasMXRecord bool
	Records     []*net.MX
}

func (v *Verifier) CheckMX(domain string) (*Mx, error) {
	records, err := net.LookupMX(domain)
	if err != nil {
		return nil, parseMXError(err)
	}

	if len(records) == 0 {
		return &Mx{HasMXRecord: false}, nil
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Pref < records[j].Pref
	})

	return &Mx{
		HasMXRecord: true,
		Records:     records,
	}, nil
}
