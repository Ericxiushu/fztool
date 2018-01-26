package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/astaxie/beego"
)

var (
	childDir = "res"
	Int      = []byte("int")
	Int64    = []byte("int64")
	String   = []byte("string")
	Float64  = []byte("float64")

	specialWords = map[string]bool{
		"id": true,
	}
)

type Field struct {
	Name    []byte
	OriName []byte
	Type    []byte
}

type FormStruct struct {
	TableName  []byte
	StructName []byte
	Fields     []Field
}

// formatSQLTemp formatSQLTemp
func formatSQLTemp(path string) error {

	sqlBody, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	sqlBody = bytes.Replace(sqlBody, []byte("\n"), []byte(""), -1)

	sqlBodyArr := bytes.Split(sqlBody, []byte(";"))
	sqlResult := []FormStruct{}
	var item FormStruct

	regTable := regexp.MustCompile("CREATE\\s*TABLE\\s*`\\w+`")
	regField := regexp.MustCompile("`\\w+`\\s*\\b\\w+")

	var tableNames [][]byte
	var fieldsReg [][]byte
	var tempFieldName []byte

	for _, v := range sqlBodyArr {
		if regTable.Match(v) {

			item = FormStruct{}

			tableNames = regTable.FindAll(v, -1)
			if len(tableNames) != 1 {
				beego.Error("[error] :", string(v), len(tableNames))
				continue
			}

			item.TableName, item.StructName = formatFieldTemp(tableNames[0])

			fieldsReg = regField.FindAll(v, -1)
			lastIndex := 0
			for _, fieldV := range fieldsReg {

				lastIndex = bytes.LastIndex(fieldV, []byte("`"))

				_, tempFieldName = formatFieldTemp(fieldV[:lastIndex+1])
				item.Fields = append(item.Fields, Field{Name: tempFieldName, OriName: fieldV[1:lastIndex], Type: checkTypeTemp(fieldV[lastIndex+1:], tempFieldName)})
			}

			sqlResult = append(sqlResult, item)
		}
	}

	for _, v := range sqlResult {

		beego.Error(string(v.TableName))

		for _, v1 := range v.Fields {
			beego.Error(string(v1.Name), string(v1.Type))
		}

	}

	writeInFileTemp(sqlResult)

	return nil
}

func formatFieldTemp(str []byte) ([]byte, []byte) {

	str = bytes.ToLower(str)
	count := len(str)

	startIndex := bytes.Index(str, []byte("`"))

	str = str[startIndex+1 : count-1]
	oriStr := str

	str = formatNames(bytes.Replace(str, []byte("_"), []byte(" "), -1))

	return oriStr, str
}

func checkTypeTemp(t []byte, fieldName []byte) []byte {

	res := String

	t = bytes.TrimSpace(t)

	switch string(t) {

	case "int":
		res = Int
		if len(fieldName) >= 4 {
			if strings.ToLower(string((fieldName[len(fieldName)-4:]))) == "time" {
				res = Int64
			}
		}
	case "float", "decimal":
		res = Float64

	}

	return res
}

func writeInFileTemp(structList []FormStruct) {

	structContent := ""
	content := ""

	for _, v := range structList {

		structContent += fmt.Sprintf("// %s %s\ntype %s struct{ \n", v.StructName, v.StructName, v.StructName)

		for _, v1 := range v.Fields {
			structContent += fmt.Sprintf("    %s %s `json:\"%s\" xorm:\"%s\"`\n", v1.Name, v1.Type, formatNames2(bytes.Replace(v1.OriName, []byte("_"), []byte(" "), -1)), v1.OriName)
		}

		structContent += "}\n\n"

		content = strings.Replace(sqlFormtField, "$structContent", structContent, -1)
		content = strings.Replace(content, "$structName", string(v.StructName), -1)
		content = strings.Replace(content, "$tableName", string(v.TableName), -1)

	}

	fileName := fmt.Sprintf("./%s/dbStruct.go", childDir)

	dirErr := os.MkdirAll(childDir, 0777)

	if dirErr != nil {
		fmt.Println("mkdir error :" + dirErr.Error())
		return
	}

	dstFile, err := os.Create(fileName)
	if err != nil {
		fmt.Println(" create file error :" + err.Error())
		return
	}
	defer dstFile.Close()
	i, e := dstFile.WriteString(content + "\n")

	if e != nil {
		fmt.Printf("ERROR: %s, %d\n", e.Error(), i)
	} else {
		fmt.Printf("SUCCESS: %d\n", i)

	}

}

func formatNames(a []byte) []byte {

	list := strings.Split(string(a), " ")

	for k, v := range list {
		if specialWords[strings.ToLower(v)] {
			list[k] = strings.ToUpper(v)
		} else {
			list[k] = strings.Title(v)
		}
	}

	return []byte(strings.Join(list, ""))
}

func formatNames2(a []byte) []byte {

	list := strings.Split(string(a), " ")

	for k, v := range list {
		if k > 0 {
			list[k] = strings.Title(v)
		}
	}

	return []byte(strings.Join(list, ""))
}

var sqlFormtField = `package models

$structContent

`

func main() {

	skillfolder := `./`
	fmt.Println(skillfolder)
	// 获取所有文件
	files, _ := ioutil.ReadDir(skillfolder)
	for _, file := range files {
		if file.IsDir() {
			continue
		} else {

			fileName := file.Name()

			if path.Ext(fileName) == ".sql" {
				// beego.Error(skillfolder + "/" + fileName)
				formatSQLTemp(skillfolder + fileName)

				fmt.Println()
				fmt.Println("/*****************/")
				fmt.Println()
				break
			}

		}
	}

	err := os.Chdir(skillfolder + childDir)
	if err != nil {
		beego.Error(err)
	}

	cmd := exec.Command("gofmt", "-w", ".")
	err = cmd.Run()
	if err != nil {
		beego.Error(err)
	}
}
