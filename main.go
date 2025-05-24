package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Symbol struct {
	Address int
	Defined bool
}

type Instruction struct {
	LineNum  int
	RawLine  string
	Label    string
	Mnemonic string
	Op1      string
	Op2      string
}

var (
	locationCounter = 0
	symbolTable     = make(map[string]Symbol)
	instructions    []Instruction
	machineCode     []byte
	listing         []string
	ended           = false
	segments        = map[string]int{}
	currentSegment  = ""
)

var opcodeMap = map[string]byte{
	"MOV": 0x88,
	"OR":  0x08,
	"MUL": 0xF6,
	"JS":  0x78,
}

var regMap = map[string]byte{
	"AL": 0x0, "CL": 0x1, "DL": 0x2, "BL": 0x3,
	"AH": 0x4, "CH": 0x5, "DH": 0x6, "BH": 0x7,
}

func trimComment(line string) string {
	if idx := strings.Index(line, ";"); idx != -1 {
		return strings.TrimSpace(line[:idx])
	}
	return strings.TrimSpace(line)
}

func tokenize(line string) []string {
	line = trimComment(line)
	tokens := []string{}
	for _, tok := range strings.Fields(line) {
		if strings.HasSuffix(tok, ":") {
			tokens = append(tokens, tok)
		} else if strings.Contains(tok, ",") {
			parts := strings.Split(tok, ",")
			tokens = append(tokens, parts[0], parts[1])
		} else {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}

func parseLine(line string, lineNum int) {
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return
	}

	inst := Instruction{LineNum: lineNum, RawLine: line}

	idx := 0
	if strings.HasSuffix(tokens[idx], ":") {
		label := strings.TrimSuffix(tokens[idx], ":")
		symbolTable[label] = Symbol{Address: locationCounter, Defined: true}
		inst.Label = label
		idx++
	}

	if idx >= len(tokens) {
		instructions = append(instructions, inst)
		return
	}

	inst.Mnemonic = strings.ToUpper(tokens[idx])
	idx++

	if idx < len(tokens) {
		inst.Op1 = strings.ToUpper(tokens[idx])
		idx++
	}
	if idx < len(tokens) {
		inst.Op2 = strings.ToUpper(tokens[idx])
	}

	// Memory reservation or instruction size
	switch inst.Mnemonic {
	case "DB":
		locationCounter += 1
	case "DW":
		locationCounter += 2
	case "ORG":
		if val, err := strconv.ParseInt(inst.Op1, 0, 64); err == nil {
			locationCounter = int(val)
		}
	case "SEGMENT":
		currentSegment = inst.Label
		segments[currentSegment] = locationCounter
	case "ENDS":
		currentSegment = ""
	case "END":
		ended = true
	default:
		// Assume 2-byte instruction for simplicity
		locationCounter += 2
	}

	instructions = append(instructions, inst)
}

func firstPass(lines []string) {
	for i, line := range lines {
		parseLine(line, i+1)
	}
}

func emit(b byte) {
	machineCode = append(machineCode, b)
}

func resolveSymbol(sym string) int {
	if val, err := strconv.ParseInt(sym, 0, 64); err == nil {
		return int(val)
	}
	if entry, ok := symbolTable[sym]; ok && entry.Defined {
		return entry.Address
	}
	fmt.Fprintf(os.Stderr, "[Error] Undefined symbol: %s\n", sym)
	return 0
}

func secondPass() {
	locationCounter = 0
	for _, inst := range instructions {
		lineOut := fmt.Sprintf("%04X  ", locationCounter)

		switch inst.Mnemonic {
		case "DB":
			val := resolveSymbol(inst.Op1)
			emit(byte(val))
			lineOut += fmt.Sprintf("%02X      %s", val&0xFF, inst.RawLine)
			locationCounter += 1

		case "DW":
			val := resolveSymbol(inst.Op1)
			emit(byte(val & 0xFF))
			emit(byte((val >> 8) & 0xFF))
			lineOut += fmt.Sprintf("%02X %02X   %s", val&0xFF, (val>>8)&0xFF, inst.RawLine)
			locationCounter += 2

		case "ORG":
			locationCounter = resolveSymbol(inst.Op1)
			lineOut = fmt.Sprintf("      ORG %s", inst.Op1)

		case "END":
			lineOut = fmt.Sprintf("      %s", inst.Mnemonic)

		case "SEGMENT", "ENDS":
			lineOut = fmt.Sprintf("      %s", inst.Mnemonic)

		case "JS":
			offset := resolveSymbol(inst.Op1) - (locationCounter + 2)
			emit(opcodeMap["JS"])
			emit(byte(offset))
			lineOut += fmt.Sprintf("%02X %02X   %s", opcodeMap["JS"], byte(offset), inst.RawLine)
			locationCounter += 2

		case "MUL":
			emit(opcodeMap["MUL"])
			modrm := 0xE0 | regMap[inst.Op1]
			emit(modrm)
			lineOut += fmt.Sprintf("%02X %02X   %s", opcodeMap["MUL"], modrm, inst.RawLine)
			locationCounter += 2

		case "MOV", "OR":
			emit(opcodeMap[inst.Mnemonic])
			reg1 := regMap[inst.Op1]
			reg2 := regMap[inst.Op2]
			modrm := 0xC0 | (reg2 << 3) | reg1
			emit(modrm)
			lineOut += fmt.Sprintf("%02X %02X   %s", opcodeMap[inst.Mnemonic], modrm, inst.RawLine)
			locationCounter += 2

		default:
			lineOut += fmt.Sprintf("      %s", inst.RawLine)
		}

		listing = append(listing, lineOut)
	}
}

func writeOutput() {
	// Save machine code
	obj, _ := os.Create("program.obj")
	defer obj.Close()
	obj.Write(machineCode)

	// Save listing
	lst, _ := os.Create("program.lst")
	defer lst.Close()
	for _, line := range listing {
		fmt.Fprintln(lst, line)
	}
}

func main() {
	file, err := os.Open("program.asm")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	firstPass(lines)
	secondPass()
	writeOutput()

	fmt.Println("Translation complete.")
}
