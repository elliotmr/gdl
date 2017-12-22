package main

import (
	"encoding/xml"
	"github.com/pkg/errors"
	"github.com/serenize/snaker"
	"strings"
	"text/template"
	"bufio"
	"bytes"
	"fmt"
)

type Description struct {
	Summary string `xml:"summary,attr"`
	Text    string `xml:",chardata"`
}

type Request struct {
	Name        string       `xml:"name,attr"`
	Type        string       `xml:"type,attr"`
	Since       string       `xml:"since,attr"`
	Description *Description `xml:"description"`
	Args        []*Arg       `xml:"arg"`
}

type Event struct {
	Name        string       `xml:"name,attr"`
	Since       string       `xml:"since,attr"`
	Description *Description `xml:"description"`
	Args        []*Arg       `xml:"arg"`
}

type Enum struct {
	Name        string       `xml:"name,attr"`
	Since       string       `xml:"since,attr"`
	Bitfield    string       `xml:"bitfield,attr"`
	Description *Description `xml:"description"`
	Entries     []*Entry     `xml:"entry"`
}

type Arg struct {
	Name        string       `xml:"name,attr"`
	Type        string       `xml:"type,attr"`
	Summary     string       `xml:"summary,attr"`
	Interface   string       `xml:"interface,attr"`
	AllowNull   string       `xml:"allow-null,attr"`
	Enum        string       `xml:"enum,attr"`
	Description *Description `xml:"description"`
}

type Entry struct {
	Name        string       `xml:"name,attr"`
	Value       string       `xml:"value,attr"`
	Summary     string       `xml:"summary,attr"`
	Since       string       `xml:"since,attr"`
	Description *Description `xml:"description"`
}

type Interface struct {
	Name        string       `xml:"name,attr"`
	Version     string       `xml:"version,attr"`
	Description *Description `xml:"description"`
	Requests    []*Request   `xml:"request"`
	Events      []*Event     `xml:"event"`
	Enums       []*Enum      `xml:"enum"`
}

type Protocol struct {
	Name        string       `xml:"name,attr"`
	Copyright   string       `xml:"copyright"`
	Description *Description `xml:"description"`
	Interfaces  []*Interface `xml:"interface"`
}

func parse(raw []byte) (*Protocol, error) {
	p := &Protocol{}
	err := xml.Unmarshal(raw, p)
	return p, errors.Wrap(err, "unable to parse xml")
}

func genTemplate(templateText string) *template.Template {
	funcMap := template.FuncMap{
		"ifname":          InterfaceName,
		"camel":           snaker.SnakeToCamel,
		"camel_lower":     snaker.SnakeToCamelLower,
		"desc_to_comment": DescriptionToComment,
		"req_sig":         ReqSignature,
		"req_ret_sig":     ReqReturnSignature,
		"req_ret":         ReqReturn,
		"evt_call":        EvtCall,
		"arg_decode":      ArgDecode,
		"arg_encode":      ArgEncode,
	}

	return template.Must(template.New("wl").Funcs(funcMap).Parse(templateText))

}

func InterfaceName(name string) string {
	name = strings.TrimLeft(name, "wl_")
	return snaker.SnakeToCamel(name)
}

func DescriptionToComment(desc string) string {
	buf := &bytes.Buffer{}
	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(desc)))
	for scanner.Scan() {
		buf.WriteString("// ")
		buf.Write(bytes.TrimSpace(scanner.Bytes()))
		buf.WriteString("\n")

	}
	return buf.String()
}

func ArgName(arg *Arg) string {
	name := snaker.SnakeToCamelLower(arg.Name)
	if name == "interface" {
		name = "iface"
	}
	return name
}

func ArgSignature(arg *Arg) string {
	name := ArgName(arg)
	buf := bytes.NewBufferString(name)
	buf.WriteString(" ")
	switch arg.Type {
	case "int":
		buf.WriteString("int32")
	case "uint", "object":
		buf.WriteString("uint32")
	case "fixed":
		buf.WriteString(("float64"))
	case "string":
		buf.WriteString("string")
	case "array":
		buf.WriteString("[]byte")
	case "fd":
		buf.WriteString("*os.File")
	case "new_id":
		if arg.Interface == "" {
			buf.WriteString("ObjectID")
		} else {
			return ""
		}
	default:
		return ""
	}
	return buf.String()
}

func ReqSignature(args []*Arg) string {
	argSigs := make([]string, 0)
	for _, arg := range args {
		newSig := ArgSignature(arg)
		if newSig != "" {
			argSigs = append(argSigs, newSig)
		}
	}
	return strings.Join(argSigs, ", ")
}

func ReqReturnSignature(args []*Arg) string {
	newTypeInterface := ""
	for _, arg := range args {
		if arg.Type == "new_id" {
			newTypeInterface = arg.Interface
			break
		}
	}
	if newTypeInterface == "" {
		return "error"
	}
	return fmt.Sprintf("(*%s, error)", InterfaceName(newTypeInterface))
}

func ReqReturn(args []*Arg) string {
	newTypeInterface := ""
	for _, arg := range args {
		if arg.Type == "new_id" {
			newTypeInterface = arg.Interface
			break
		}
	}
	if newTypeInterface == "" {
		return "nil"
	}
	return "ret, nil"
}

func EvtCall(args []*Arg) string {
	argSigs := make([]string, 0)
	for _, arg := range args {
		name := ArgName(arg)
		if name != "" && arg.Type != "new_id"{
			argSigs = append(argSigs, name)
		}
	}
	return strings.Join(argSigs, ", ")

}

func ArgDecode(args []*Arg) string {
	buf := &bytes.Buffer{}
	loc := 0
	for _, arg := range args {
		switch arg.Type {
		case "int":
			buf.WriteString(
				fmt.Sprintf(
					"\t\t\t%s := int32(hostByteOrder.Uint32(payload[%d+off:%d+off]))\n",
					ArgName(arg),
					loc,
					loc+4,
				),
			)
			loc += 4
		case "uint", "object":
			buf.WriteString(
				fmt.Sprintf(
					"\t\t\t%s := hostByteOrder.Uint32(payload[%d+off:%d+off])\n",
					ArgName(arg),
					loc,
					loc+4,
				),
			)
			loc += 4
		case "new_id":

			loc += 4
		case "fixed":
			buf.WriteString(
				fmt.Sprintf(
					"\t\t\t%s := fixedToFloat64(int32(hostByteOrder.Uint32(payload[%d+off:%d+off])))\n",
					ArgName(arg),
					loc,
					loc+4,
				),
			)
			loc += 4
		case "string":
			buf.WriteString(
				fmt.Sprintf(
					"\t\t\tlen = int(hostByteOrder.Uint32(payload[%d+off:%d+off]))\n",
					loc,
					loc+4,
				),
			)
			loc += 4
			buf.WriteString(
				fmt.Sprintf(
					"\t\t\t%s := string(payload[%d+off:%d+len+off-1])\n",
					ArgName(arg),
					loc,
					loc,
				),
			)
			buf.WriteString("\t\t\toff += len\n")
			buf.WriteString("\t\t\tif off % 4 != 0 {\n")
			buf.WriteString("\t\t\t\toff += 4 - off % 4\n")
			buf.WriteString("\t\t\t}\n")
		case "array":
			buf.WriteString(
				fmt.Sprintf(
					"\t\t\tlen = int(hostByteOrder.Uint32(payload[%d+off:%d+off]))\n",
					loc,
					loc+4,
				),
			)
			loc += 4
			buf.WriteString(
				fmt.Sprintf(
					"\t\t\t%s := payload[%d+off:%d+len+off]\n",
					ArgName(arg),
					loc,
					loc,
				),
			)
			buf.WriteString("\t\t\toff += len\n")
			buf.WriteString("\t\t\tif off % 4 != 0 {\n")
			buf.WriteString("\t\t\t\toff += 4 - off % 4\n")
			buf.WriteString("\t\t\t}\n")
		case "fd":
			buf.WriteString(fmt.Sprintf("\t\t\t%s := file\n", ArgName(arg)))
		default:
			return ""
		}
	}
	return buf.String()
}

func ArgEncode(args []*Arg) string {
	buf := &bytes.Buffer{}
	for _, arg := range args {
		switch arg.Type {
		case "int", "uint", "object":
			buf.WriteString(
				fmt.Sprintf(
					"\tbinary.Write(this.c.buf, hostByteOrder, %s)\n",
					ArgName(arg),
				),
			)
		case "new_id":
			if arg.Interface == "" {
				buf.WriteString(
					fmt.Sprintf(
						"\tbinary.Write(this.c.buf, hostByteOrder, %s)\n",
						ArgName(arg),
					),
				)
			} else {
				buf.WriteString(
					fmt.Sprintf(
						"\tret := this.c.New%s()\n",
						InterfaceName(arg.Interface),
					),
				)
				buf.WriteString("\tbinary.Write(this.c.buf, hostByteOrder, ret.ID())\n")

			}
		case "fixed":
			buf.WriteString(
				fmt.Sprintf(
					"\ttmp = float64ToFixed(%s)\n",
					ArgName(arg),
				),
			)
			buf.WriteString("\tbinary.Write(this.c.buf, hostByteOder, tmp)\n")
		case "string":
			buf.WriteString(
				fmt.Sprintf(
					"\tbinary.Write(this.c.buf, hostByteOrder, len(%s))\n",
					ArgName(arg),
				),
			)
			buf.WriteString(
				fmt.Sprintf(
					"\tthis.c.buf.WriteString(%s)\n",
					ArgName(arg),
				),
			)
		case "array":
			buf.WriteString(
				fmt.Sprintf(
					"\tbinary.Write(this.c.buf, hostByteOrder, len(%s))\n",
					ArgName(arg),
				),
			)
			buf.WriteString(
				fmt.Sprintf(
					"\tthis.c.buf.Write(%s)\n",
					ArgName(arg),
				),
			)
		case "fd":
			buf.WriteString(fmt.Sprintf("\t// TODO: handle fds\n"))
		default:
			return ""
		}

	}
	return buf.String()
}

func main() {

}
