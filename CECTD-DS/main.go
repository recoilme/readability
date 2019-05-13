package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

type tree struct {
	key        string
	parent     string
	node       *html.Node
	density    *density
	densitySum int
}

type density struct {
	chars    int
	tags     int
	linkchar int
	linktag  int
	txt      string
}

func main() {
	f := readFile("html/4.htm")

	parse(f)
	defer f.Close()
	calcDensity(treeMap)
	calcDensitySum(treeMap)
	//calcChild(treeMap)
	candidat := printTree(treeMap)
	printCandidat(candidat)
}

func (d *density) density() int {
	if d.tags <= 0 {
		return (d.chars)
	}
	return int(float32(d.chars) / float32(d.tags))
}

func (d *density) textDensity() int {
	//text_density = (1.0 * char_num / tag_num) * qLn((1.0 * char_num * tag_num) / (1.0 * linkchar_num * linktag_num))
	// / qLn(qLn(1.0 * char_num * linkchar_num / un_linkchar_num + ratio * char_num + qExp(1.0)));
	//return 0

	if d.chars == 0 {
		return 0
	}
	unlinkcharnum := d.chars - d.linkchar
	if d.tags <= 0 {
		d.tags = 1
	}
	if d.linkchar <= 0 {
		d.linkchar = 1
	}
	if d.linktag <= 0 {
		d.linktag = 1
	}
	if unlinkcharnum <= 0 {
		unlinkcharnum = 1
	}
	chisl := (float64(d.chars) / float64(d.tags)) * math.Log((float64(d.chars)*float64(d.tags))/(float64(d.linkchar)*float64(d.linktag)))
	//qLn(qLn(1.0*char_num*linkchar_num/un_linkchar_num + ratio*char_num + qExp(1.0)))
	znamen := math.Log( /*math.Log*/ (float64(d.chars)*float64(d.linkchar)/float64(unlinkcharnum) + 1.0*float64(d.chars) + math.Exp(1.0)))
	return int(chisl / znamen)
}

func (d *density) score() int {

	if d.tags <= 0 {
		d.tags = 1
	}
	if d.linkchar <= 0 {
		d.linkchar = 1
	}
	if d.chars <= 0 {
		d.chars = 1
	}
	//log.Println("log", math.Log2(float64(d.chars)/float64(d.hyper)), d.chars, d.hyper)
	score := (float64(d.chars) / float64(d.tags)) * math.Log2(float64(d.chars)/float64(d.linkchar))
	return int(score)
}

var (
	treeMap = make(map[string]*tree)
)

func calcDensitySum(t map[string]*tree) {
	for _, v := range treeMap {
		v.densitySum = calcSum(v)
	}
}

func calcSum(t *tree) int {
	sum := t.density.textDensity()
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			key := nodeHash(n)
			if v, ok := treeMap[key]; ok {
				sum += v.density.textDensity()
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(t.node)
	return sum
}

func calcDensity(t map[string]*tree) {
	for _, v := range treeMap {
		v.density = calcChars(v.node)
		//log.Println(v.key, "chars:", ch, "hyper:", hyper, "tags:", tags, "dencity:", d.score())
	}
}

func calcChars(n *html.Node) *density {
	txt := ""
	tags := -1
	linkchar := ""
	linktag := 0
	var f func(*html.Node, bool, bool)
	f = func(n *html.Node, isText, isHyper bool) {

		if n.Type == html.ElementNode {
			if isHyper {
				linktag++
			} else {
				tags++
			}

		}
		if n.Type == html.TextNode {

			if isHyper {
				linkchar += strings.TrimSpace(n.Data)
			} else {
				if isText {
					txt += strings.TrimSpace(n.Data)
				}
			}

		}
		isText = isText || (n.Type == html.ElementNode)
		isHyper = isHyper || (n.Type == html.ElementNode && n.Data == "a")
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c, isText, isHyper)
		}
	}

	f(n, false, false)
	txt = strClean(txt)
	linkchar = strClean(linkchar)
	d := &density{chars: len(txt), tags: tags, linkchar: len(linkchar), linktag: linktag, txt: txt}
	return d
}

func strClean(txt string) string {
	txt = strings.Replace(txt, "\r\n", "", -1)
	txt = strings.Replace(txt, "\n", "", -1)
	txt = strings.TrimSpace(txt)
	return txt
}

func printTree(t map[string]*tree) string {
	sl := make([]*tree, 0)
	for _, t := range treeMap {
		sl = append(sl, t)
		//log.Println(v.key, "parent:", v.parent)
		//log.Println("child:")
		//for _, c := range v.child {
		//	log.Println(c.key)
		//}
		//log.Println("----")
	}
	sort.Slice(sl, func(i, j int) bool {
		return sl[i].densitySum > sl[j].densitySum
	})
	candidat := ""
	for i, t := range sl {
		max := 50
		if len(t.key) < max {
			max = len(t.key)
		}
		//log.Println(t.key[:max], t.density.textDensity(), "sum:", t.densitySum, "chars:", t.density.chars)
		if i > 9 {
			break
		}
		if !strings.HasPrefix(t.key, "body") {
			candidat = t.key
			break
		}
	}
	return candidat
}

func trimLongStr(s string) string {
	max := 30
	if len(s) < max {
		max = len(s)
	}
	return s[:max]
}

func readFile(p string) *os.File {
	file, err := os.Open(p)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func removeNode(n *html.Node, bad map[string]struct{}) bool {
	var r bool
	// if note is script tag
	if n.Type == html.ElementNode {
		atom := strings.ToLower(n.Data)
		if _, ok := bad[atom]; ok {
			n.Parent.RemoveChild(n)
			return true
		}
	}
	if n.Type == html.CommentNode {
		n.Parent.RemoveChild(n)
		return true
	}
	// traverse DOM
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		r = removeNode(c, bad)
		if r {
			break
		}
	}
	return r
}

func removeBad(n *html.Node) {
	bad := map[string]struct{}{
		"style":    struct{}{},
		"script":   struct{}{},
		"svg":      struct{}{},
		"nav":      struct{}{},
		"aside":    struct{}{},
		"form":     struct{}{},
		"noscript": struct{}{},
		"xmp":      struct{}{},
		"textarea": struct{}{},
		"air":      struct{}{},
	}
	for {
		if !removeNode(n, bad) {
			break
		}
	}
}

func parse(r io.Reader) {
	node, err := html.Parse(r)
	if err != nil {
		log.Fatal(err)
	}
	removeBad(node)
	var f func(*html.Node)
	inBody := false
	f = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if !inBody && c.Type == html.ElementNode && strings.ToLower(c.Data) == "body" {
				inBody = true
			}
			if inBody && c.Type == html.ElementNode {
				parent := ""
				key := nodeHash(c)
				//log.Println(key)
				if c.Parent != nil && c.Parent.Type == html.ElementNode {
					parent = nodeHash(c.Parent)
				}
				t := &tree{key: key, parent: parent, node: c}
				treeMap[key] = t
			}
			f(c)
		}
	}
	f(node)
}

func nodeHash(node *html.Node) (s string) {
	if node.Type == html.ElementNode {
		s += node.Data
		for _, a := range node.Attr {
			s += fmt.Sprintf(" %s %s", a.Key, a.Val)
		}
	}
	return strings.ToLower(s)
}

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}

func printCandidat(c string) {
	//cnt := 0
	if t, ok := treeMap[c]; ok {
		threshold := int(float32(t.densitySum) * 0.2)
		currents := make(map[string]bool)
		tag := ""
		//print := func(c, k, p string) {
		//log.Println("cur:", trimLongStr(c), "key:", trimLongStr(k), "par:", trimLongStr(p))
		//}
		var f func(*html.Node, bool)
		f = func(n *html.Node, printText bool) {
			if n.Type == html.ElementNode {
				key := nodeHash(n)
				if v, ok := treeMap[key]; ok {
					tag = n.Data
					if v.densitySum > threshold {
						//printText = true
						currents[v.key] = true
						//current = v.key
						//cnt++
						//print("current", v.key, v.parent)

						//log.Println(trimLongStr(v.key), "par:", trimLongStr(v.parent)) //, v.densitySum, v.density.textDensity())
						//log.Println(v.density.txt)
					} else {
						//printText = false
						if _, ok := currents[v.parent]; ok {
							//printText = true
							//print("current", v.key, v.parent)
							//current = v.key
							currents[v.key] = true
						}
					}
				}
				if _, ok := currents[key]; ok {
					printText = true

				} else {
					printText = false
				}
			}

			if n.Type == html.TextNode && printText {
				d := n.Data
				space := regexp.MustCompile(`\s+`)
				s := space.ReplaceAllString(d, " ")
				if s != " " {
					fmt.Printf("%s:%s\n", tag, s)
				}

			}
			//printText = printText || (n.Type == html.ElementNode && n.Data == "div")

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				/*
					if cnt > 100 {
						break
					}*/
				f(c, printText)
			}
		}
		f(t.node, false)
	}
}
