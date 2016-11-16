package kumo

import (
	"regexp"

	"github.com/sww/dumblog"
)

type Filter struct {
	Logger  *dumblog.DumbLog
	regexps []*regexp.Regexp
}

func NewFilter(regexps ...string) *Filter {
	var s []*regexp.Regexp
	for _, r := range regexps {
		s = append(s, regexp.MustCompile(r))
	}

	return &Filter{regexps: s}
}

func (f *Filter) HasFilters() bool {
	return len(f.regexps) > 0
}

// Returns whether or not the string s matches Filter.regexps.
func (f *Filter) Filter(s string) bool {
	for _, re := range f.regexps {
		if re.MatchString(s) {
			f.Logger.Printf("Regexp %v Matched %s", re, s)
			return true
		}
	}

	return false
}

func (f *Filter) FilterNzb(nzb *NZB) *NZB {
	if len(f.regexps) < 1 {
		return nzb
	}

	var filteredNzb NZB
	for _, file := range nzb.Files {
		if !f.Filter(file.Subject) {
			filteredNzb.Files = append(filteredNzb.Files, file)
		}
	}

	return &filteredNzb
}
