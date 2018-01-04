package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/serenize/snaker"
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

var T *template.Template
var D map[string]string

func genTemplate(templateText string) {
	funcMap := template.FuncMap{
		"camel":           snaker.SnakeToCamel,
		"camel_lower":     snaker.SnakeToCamelLower,
		"get":             GetGlobal,
		"set":             SetGlobal,
		"ifname":          InterfaceName,
		"arg_name":        ArgName,
		"desc_to_comment": DescriptionToComment,
		"is_constructor":  IsConstructor,
		"req_sig":         ReqSignature,
		"req_ret_sig":     ReqReturnSignature,
		"req_ret":         ReqReturn,
		"evt_call":        EvtCall,
		"arg_decode":      ArgDecode,
		"arg_encode":      ArgEncode,
	}
	D = make(map[string]string)
	T = template.Must(template.New("wl").Funcs(funcMap).Parse(templateText))
}

func GetGlobal(key string) string {
	return D[key]
}

func SetGlobal(key string, val string) string {
	D[key] = val
	return ""
}

func InterfaceName(name string) string {
	name = strings.TrimLeft(name, "wl_")
	return snaker.SnakeToCamel(name)
}

func ArgName(arg *Arg) string {
	name := snaker.SnakeToCamelLower(arg.Name)
	if name == "interface" {
		name = "iface"
	}
	return name
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

func argSignature(arg *Arg) string {
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
			buf.WriteString("uint32")
		} else {
			buf.Reset()
			buf.WriteString("l ")
			buf.WriteString(InterfaceName(arg.Interface))
			buf.WriteString("Listener")
		}
	default:
		return ""
	}
	return buf.String()
}

func ReqSignature(args []*Arg) string {
	argSigs := make([]string, 0)
	for _, arg := range args {
		// Special case for bind
		if arg.Type == "new_id" && arg.Interface == "" {
			argSigs = append(argSigs, "iface string")
			argSigs = append(argSigs, "version uint32")
		}
		newSig := argSignature(arg)
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

func IsConstructor(args []*Arg) bool {
	for _, arg := range args {
		if arg.Type == "new_id" && arg.Interface != "" {
			return true
		}
	}
	return false
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
		if name != "" && arg.Type != "new_id" {
			argSigs = append(argSigs, name)
		}
	}
	return strings.Join(argSigs, ", ")

}

/*
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
*/

func ArgDecode(args []*Arg) string {
	buf := &bytes.Buffer{}
	for _, arg := range args {
		switch arg.Type {
		case "int":
			T.ExecuteTemplate(buf, "decode-int", ArgName(arg))
		case "uint", "object":
			T.ExecuteTemplate(buf, "decode-uint", ArgName(arg))
		case "new_id":
			// TODO: implement
		case "fixed":
			T.ExecuteTemplate(buf, "decode-fixed", ArgName(arg))
		case "string":
			T.ExecuteTemplate(buf, "decode-string", ArgName(arg))
		case "array":
			T.ExecuteTemplate(buf, "decode-array", ArgName(arg))
		case "fd":
			T.ExecuteTemplate(buf, "decode-fd", ArgName(arg))
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
			T.ExecuteTemplate(buf, "encode-base", ArgName(arg))
		case "new_id":
			// Special case handling for "bind"
			if arg.Interface == "" {
				T.ExecuteTemplate(buf, "encode-bind", ArgName(arg))
			} else {
				T.ExecuteTemplate(buf, "encode-new-id", InterfaceName(arg.Interface))
			}
		case "fixed":
			T.ExecuteTemplate(buf, "encode-fixed", ArgName(arg))
		case "string":
			T.ExecuteTemplate(buf, "encode-string", ArgName(arg))
		case "array":
			T.ExecuteTemplate(buf, "encode-array", ArgName(arg))
		case "fd":
			T.ExecuteTemplate(buf, "encode-fd", ArgName(arg))
		default:
			return ""
		}
	}
	return buf.String()
}

func main() {

}
