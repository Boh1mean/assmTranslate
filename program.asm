SEGMENT data
value1: DB 10        ; байтовая переменная
value2: DW 1234h     ; словарная переменная
ENDS

SEGMENT code
ORG 100h

start:  MOV AX, value2   ; AX ← [value2]
        MOV BX, AX       ; BX ← AX
        OR  AL, BL       ; Побитовое ИЛИ AL и BL
        MUL AL           ; AX ← AL * AL
        JS  error        ; Прыжок, если знак

        MOV CX, [SI+4]   ; CX ← [SI+4]
        MOV [SI+6], AX   ; [SI+6] ← AX

        MOV AX, 2
        MOV [value1], AL ; [value1] ← 2

        JP done

error:  MOV AX, 0FFFFh   ; код ошибки

done:   INT 21h          ; выход в DOS

ENDS
END
