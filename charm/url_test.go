package charm_test

import (
	"encoding/json"
	"fmt"
	"labix.org/v2/mgo/bson"
	. "launchpad.net/gocheck"
	"launchpad.net/juju-core/charm"
	"regexp"
)

type URLSuite struct{}

var _ = Suite(&URLSuite{})

var urlTests = []struct {
	s, err string
	url    *charm.URL
}{
	{"cs:~user/series/name", "", &charm.URL{"cs", "user", "series", "name", -1}},
	{"cs:~user/series/name-0", "", &charm.URL{"cs", "user", "series", "name", 0}},
	{"cs:series/name", "", &charm.URL{"cs", "", "series", "name", -1}},
	{"cs:series/name-42", "", &charm.URL{"cs", "", "series", "name", 42}},
	{"local:series/name-1", "", &charm.URL{"local", "", "series", "name", 1}},
	{"local:series/name", "", &charm.URL{"local", "", "series", "name", -1}},
	{"local:series/n0-0n-n0", "", &charm.URL{"local", "", "series", "n0-0n-n0", -1}},

	{"bs:~user/series/name-1", "charm URL has invalid schema: .*", nil},
	{"cs:~1/series/name-1", "charm URL has invalid user name: .*", nil},
	{"cs:~user/1/name-1", "charm URL has invalid series: .*", nil},
	{"cs:~user/series/name-1-2", "charm URL has invalid charm name: .*", nil},
	{"cs:~user/series/name-1-name-2", "charm URL has invalid charm name: .*", nil},
	{"cs:~user/series/name--name-2", "charm URL has invalid charm name: .*", nil},
	{"cs:~user/series/huh/name-1", "charm URL has invalid form: .*", nil},
	{"cs:~user/name", "charm URL without series: .*", nil},
	{"cs:name", "charm URL without series: .*", nil},
	{"local:~user/series/name", "local charm URL with user name: .*", nil},
	{"local:~user/name", "local charm URL with user name: .*", nil},
	{"local:name", "charm URL without series: .*", nil},
}

func (s *URLSuite) TestParseURL(c *C) {
	for i, t := range urlTests {
		c.Logf("test %d", i)
		url, err := charm.ParseURL(t.s)
		comment := Commentf("ParseURL(%q)", t.s)
		if t.err != "" {
			c.Check(err.Error(), Matches, t.err, comment)
		} else {
			c.Check(url, DeepEquals, t.url, comment)
			c.Check(t.url.String(), Equals, t.s)
		}
	}
}

var inferTests = []struct {
	vague, exact string
}{
	{"foo", "cs:defseries/foo"},
	{"foo-1", "cs:defseries/foo-1"},
	{"n0-n0-n0", "cs:defseries/n0-n0-n0"},
	{"cs:foo", "cs:defseries/foo"},
	{"local:foo", "local:defseries/foo"},
	{"series/foo", "cs:series/foo"},
	{"cs:series/foo", "cs:series/foo"},
	{"local:series/foo", "local:series/foo"},
	{"cs:~user/foo", "cs:~user/defseries/foo"},
	{"cs:~user/series/foo", "cs:~user/series/foo"},
	{"local:~user/series/foo", "local:~user/series/foo"},
	{"bs:foo", "bs:defseries/foo"},
	{"cs:~1/foo", "cs:~1/defseries/foo"},
	{"cs:foo-1-2", "cs:defseries/foo-1-2"},
}

func (s *URLSuite) TestInferURL(c *C) {
	for i, t := range inferTests {
		c.Logf("test %d", i)
		comment := Commentf("InferURL(%q, %q)", t.vague, "defseries")
		inferred, ierr := charm.InferURL(t.vague, "defseries")
		parsed, perr := charm.ParseURL(t.exact)
		if parsed != nil {
			c.Check(inferred, DeepEquals, parsed, comment)
		} else {
			expect := perr.Error()
			if t.vague != t.exact {
				expect = fmt.Sprintf("%s (URL inferred from %q)", expect, t.vague)
			}
			c.Check(ierr.Error(), Equals, expect, comment)
		}
	}
	u, err := charm.InferURL("~blah", "defseries")
	c.Assert(u, IsNil)
	c.Assert(err, ErrorMatches, "cannot infer charm URL with user but no schema: .*")
}

var validRegexpTests = []struct {
	regexp *regexp.Regexp
	string string
	expect bool
}{
	{charm.ValidUser, "", false},
	{charm.ValidUser, "bob", true},
	{charm.ValidUser, "Bob", false},
	{charm.ValidUser, "bOB", true},
	{charm.ValidUser, "b^b", false},
	{charm.ValidUser, "bob1", true},
	{charm.ValidUser, "bob-1", true},
	{charm.ValidUser, "bob+1", true},
	{charm.ValidUser, "bob.1", true},
	{charm.ValidUser, "1bob", true},
	{charm.ValidUser, "1-bob", true},
	{charm.ValidUser, "1+bob", true},
	{charm.ValidUser, "1.bob", true},
	{charm.ValidUser, "jim.bob+99-1.", true},

	{charm.ValidName, "", false},
	{charm.ValidName, "wordpress", true},
	{charm.ValidName, "Wordpress", false},
	{charm.ValidName, "word-press", true},
	{charm.ValidName, "word press", false},
	{charm.ValidName, "word^press", false},
	{charm.ValidName, "-wordpress", false},
	{charm.ValidName, "wordpress-", false},
	{charm.ValidName, "wordpress2", true},
	{charm.ValidName, "wordpress-2", false},
	{charm.ValidName, "word2-press2", true},

	{charm.ValidSeries, "", false},
	{charm.ValidSeries, "precise", true},
	{charm.ValidSeries, "Precise", false},
	{charm.ValidSeries, "pre cise", false},
	{charm.ValidSeries, "pre-cise", true},
	{charm.ValidSeries, "pre^cise", false},
	{charm.ValidSeries, "prec1se", false},
	{charm.ValidSeries, "-precise", false},
	{charm.ValidSeries, "precise-", false},
	{charm.ValidSeries, "pre-c1se", false},
}

func (s *URLSuite) TestValidRegexps(c *C) {
	for i, t := range validRegexpTests {
		c.Logf("test %d: %s", i, t.string)
		c.Assert(t.regexp.MatchString(t.string), Equals, t.expect)
	}
}

func (s *URLSuite) TestMustParseURL(c *C) {
	url := charm.MustParseURL("cs:series/name")
	c.Assert(url, DeepEquals, &charm.URL{"cs", "", "series", "name", -1})
	f := func() { charm.MustParseURL("local:name") }
	c.Assert(f, PanicMatches, "charm URL without series: .*")
}

func (s *URLSuite) TestWithRevision(c *C) {
	url := charm.MustParseURL("cs:series/name")
	other := url.WithRevision(1)
	c.Assert(url, DeepEquals, &charm.URL{"cs", "", "series", "name", -1})
	c.Assert(other, DeepEquals, &charm.URL{"cs", "", "series", "name", 1})

	// Should always copy. The opposite behavior is error prone.
	c.Assert(other.WithRevision(1), Not(Equals), other)
	c.Assert(other.WithRevision(1), DeepEquals, other)
}

var codecs = []struct {
	Marshal   func(interface{}) ([]byte, error)
	Unmarshal func([]byte, interface{}) error
}{{
	Marshal:   bson.Marshal,
	Unmarshal: bson.Unmarshal,
}, {
	Marshal:   json.Marshal,
	Unmarshal: json.Unmarshal,
}}

func (s *URLSuite) TestCodecs(c *C) {
	for i, codec := range codecs {
		c.Logf("codec %d", i)
		type doc struct {
			URL *charm.URL
		}
		url := charm.MustParseURL("cs:series/name")
		data, err := codec.Marshal(doc{url})
		c.Assert(err, IsNil)
		var v doc
		err = codec.Unmarshal(data, &v)
		c.Assert(v.URL, DeepEquals, url)

		data, err = codec.Marshal(doc{})
		c.Assert(err, IsNil)
		err = codec.Unmarshal(data, &v)
		c.Assert(err, IsNil)
		c.Assert(v.URL, IsNil)
	}
}

type QuoteSuite struct{}

var _ = Suite(&QuoteSuite{})

func (s *QuoteSuite) TestUnmodified(c *C) {
	// Check that a string containing only valid
	// chars stays unmodified.
	in := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-"
	out := charm.Quote(in)
	c.Assert(out, Equals, in)
}

func (s *QuoteSuite) TestQuote(c *C) {
	// Check that invalid chars are translated correctly.
	in := "hello_there/how'are~you-today.sir"
	out := charm.Quote(in)
	c.Assert(out, Equals, "hello_5f_there_2f_how_27_are_7e_you-today.sir")
}
