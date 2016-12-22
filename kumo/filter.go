package kumo

import (
	"regexp"
	"strings"

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
			f.Logger.Printf("[FILTER] Regexp %v Matched %q", re, s)
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

func (f *Filter) Split(nzb *NZB, extension string) []*NZB {
	var filtered NZB
	var unfiltered NZB
	for _, file := range nzb.Files {
		if strings.EqualFold(file.Extension(), extension) {
			f.Logger.Printf("[FILTER] Filtered out extension %q from subject %q", extension, file.Subject)
			filtered.Files = append(filtered.Files, file)
		} else {
			unfiltered.Files = append(unfiltered.Files, file)
		}
	}
	return []*NZB{&unfiltered, &filtered}
}
