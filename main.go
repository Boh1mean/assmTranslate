package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Instruction struct {
	LineNum     int
	Address     int
	Label       string
	Mnemonic    string
	Op1         string
	Op2         string
	RawLine     string
	MachineCode string
}

var (
	instructions    []Instruction
	symbolTable     = make(map[string]int)
	locationCounter = 0x0000
	segmentOrigin   = 0x0000
	lineNumber      = 0
)

func main() {
	inputFile := "program.asm"
	readSource(inputFile)
	firstPass()
	secondPass()
	writeListing()
}

func readSource(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		parsed := parseLine(line)
		parsed.LineNum = lineNumber
		parsed.RawLine = line
		instructions = append(instructions, parsed)
	}
}

func parseLine(line string) Instruction {
	parts := strings.Fields(line)
	var inst Instruction

	if len(parts) == 0 {
		return inst
	}

	if strings.HasSuffix(parts[0], ":") {
		inst.Label = strings.TrimSuffix(parts[0], ":")
		parts = parts[1:]
	}

	if len(parts) > 0 {
		inst.Mnemonic = strings.ToUpper(parts[0])
	}
	if len(parts) > 1 {
		op := strings.Split(parts[1], ",")
		inst.Op1 = strings.TrimSpace(op[0])
		if len(op) > 1 {
			inst.Op2 = strings.TrimSpace(op[1])
		}
	}

	return inst
}

func firstPass() {
	locationCounter = 0
	segmentOrigin = 0

	for i, inst := range instructions {
		switch inst.Mnemonic {
		case "SEGMENT":
			locationCounter = 0
			segmentOrigin = 0
			instructions[i].Address = locationCounter

		case "ORG":
			valStr := strings.TrimSuffix(strings.TrimPrefix(inst.Op1, "0x"), "h")
			val, _ := strconv.ParseInt(valStr, 16, 16)
			locationCounter = int(val)
			segmentOrigin = locationCounter
			instructions[i].Address = locationCounter

		case "DB":
			instructions[i].Address = locationCounter
			if inst.Label != "" {
				symbolTable[inst.Label] = locationCounter
			}
			locationCounter += 1

		case "DW":
			instructions[i].Address = locationCounter
			if inst.Label != "" {
				symbolTable[inst.Label] = locationCounter
			}
			locationCounter += 2

		case "ENDS", "END":
			instructions[i].Address = locationCounter

		default:
			instructions[i].Address = locationCounter
			if inst.Label != "" {
				symbolTable[inst.Label] = locationCounter
			}
			locationCounter += 2
		}
	}
}

func secondPass() {
	locationCounter = segmentOrigin

	for i, inst := range instructions {
		var code []byte

		switch inst.Mnemonic {
		case "MOV":
			code = []byte{0x88, 0xC0} // Примерный код
		case "OR":
			code = []byte{0x08, 0xC0}
		case "MUL":
			code = []byte{0xF6, 0xE0}
		case "JS":
			offset := symbolTable[inst.Op1] - (locationCounter + 2)
			code = []byte{0x78, byte(offset)}
		case "JP":
			offset := symbolTable[inst.Op1] - (locationCounter + 2)
			code = []byte{0x7A, byte(offset)}
		case "DB":
			val, _ := strconv.Atoi(inst.Op1)
			code = []byte{byte(val)}
		case "DW":
			val, _ := strconv.ParseInt(inst.Op1, 0, 16)
			code = []byte{byte(val & 0xFF), byte((val >> 8) & 0xFF)}
		default:
			// Не генерирует код
		}

		machineCodeStr := ""
		for _, b := range code {
			machineCodeStr += fmt.Sprintf("%02X", b)
		}
		instructions[i].MachineCode = machineCodeStr
		instructions[i].Address = locationCounter
		locationCounter += len(code)
	}
}

func writeListing() {
	out, _ := os.Create("output.txt")
	defer out.Close()

	fmt.Fprintln(out, strings.Repeat("=", 95))
	fmt.Fprintf(out, "[LINE]  LOC   MACHINE CODE     %-10s%s\n", "LABEL", "SOURCE")
	fmt.Fprintln(out, strings.Repeat("=", 95))

	for _, inst := range instructions {
		label := ""
		if inst.Label != "" {
			label = inst.Label + ":"
		}

		loc := ""
		if inst.MachineCode != "" || inst.Mnemonic == "ORG" || inst.Mnemonic == "SEGMENT" || inst.Mnemonic == "DW" || inst.Mnemonic == "DB" {
			loc = fmt.Sprintf("%04X", inst.Address)
		}

		line := fmt.Sprintf("[%-3d]  %-4s  %-13s  %-10s  %s",
			inst.LineNum,
			loc,
			inst.MachineCode,
			label,
			inst.RawLine)
		fmt.Fprintln(out, line)
	}
}
