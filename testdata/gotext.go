// This file is used to test the default gotext functions
package testdata

import "github.com/leonelquinteros/gotext"

func x() {
	gotext.Get("String from gotext package")

	l := gotext.NewLocale("", "")
	l.Get("String from gotext.Locale")

	po := gotext.NewPo()
	po.Get("String from gotext.Po")

	mo := gotext.NewMo()
	mo.Get("String from gotext.Mo")
}
