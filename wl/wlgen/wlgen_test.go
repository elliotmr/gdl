package main

import (
	"testing"
	"io/ioutil"
	"github.com/stretchr/testify/assert"
	"os"
	"bytes"
	"go/format"
)

func TestParse(t *testing.T) {
	data, err := ioutil.ReadFile("xdg-shell-unstable-v6.xml")
	assert.NoError(t, err)
	p, err := parse(data)
	assert.NoError(t, err)
	for _, i := range p.Interfaces {
		t.Logf("Interface Parsed: %s", i.Name)
		for _, rq := range i.Requests {
			t.Logf("  Request: %s", rq.Name)
			for _, arg := range rq.Args {
				t.Logf("    Arg: %s - %s", arg.Name, arg.Type)
			}
		}
		for _, ev := range i.Events {
			t.Logf("  Event: %s", ev.Name)
			for _, arg := range ev.Args {
				t.Logf("    Arg: %s - %s", arg.Name, arg.Type)
			}
		}
		for _, en := range i.Enums {
			t.Logf("  Enum: %s", en.Name)
		}
	}
}

func TestGenTemplate(t *testing.T) {
	data, err := ioutil.ReadFile("wayland.xml")
	assert.NoError(t, err)
	p, err := parse(data)
	assert.NoError(t, err)
	tmplText, err := ioutil.ReadFile("wl.gotmpl")
	assert.NoError(t, err)
	genTemplate(string(tmplText))
	f, err := os.Create("../protocol.go")
	assert.NoError(t, err)
	T.ExecuteTemplate(f, "base", p)
}

func TestGenTemplateWLP(t *testing.T) {
	data, err := ioutil.ReadFile("wayland.xml")
	assert.NoError(t, err)
	p, err := parse(data)
	assert.NoError(t, err)
	tmplText, err := ioutil.ReadFile("wlp.gotmpl")
	assert.NoError(t, err)
	genTemplate(string(tmplText))
	buf := &bytes.Buffer{}
	assert.NoError(t, T.ExecuteTemplate(buf, "root", p))
	f, err := os.Create("../wlp/wayland.go")
	assert.NoError(t, err)
	//f.Write(buf.Bytes())
	out, err := format.Source(buf.Bytes())
	assert.NoError(t, err)
	f.Write(out)
}

func TestGenTemplateXDG(t *testing.T) {
	data, err := ioutil.ReadFile("xdg-shell-unstable-v6.xml")
	assert.NoError(t, err)
	p, err := parse(data)
	assert.NoError(t, err)
	tmplText, err := ioutil.ReadFile("wlp.gotmpl")
	assert.NoError(t, err)
	genTemplate(string(tmplText))
	buf := &bytes.Buffer{}
	assert.NoError(t, T.ExecuteTemplate(buf, "root", p))
	f, err := os.Create("../wlp/xdg-shell.go")
	assert.NoError(t, err)
	//f.Write(buf.Bytes())
	out, err := format.Source(buf.Bytes())
	assert.NoError(t, err)
	f.Write(out)
}
