package main

import "github.com/pinpt/ripsrc/cmd"

func main() {
	// d := diffmatchpatch.New()
	// o, _ := ioutil.ReadFile("out")
	// p, err := d.PatchFromText(string(o))
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(p)
	cmd.Execute()
}
