SEGMENT code
ORG 0x100
start:  MOV AL, CL
        OR AL, BL
        MUL AL
        JS start
value:  DB 42
word:   DW 0x1234
ENDS code
END
