package migrations

import (
	"fmt"
	"regexp"
	"strings"
)

var dollarQuote = regexp.MustCompile(`^\$(?:[A-Za-z_][A-Za-z0-9_]*)?\$`)

// Remove literals/comments before inspecting statement starts. Function bodies may contain BEGIN/END.
// Migrations are trusted owner-written SQL, not an untrusted SQL sandbox.
func transactionFree(sql string) error {
	var plain strings.Builder
	for i := 0; i < len(sql); {
		switch {
		case strings.HasPrefix(sql[i:], "--"):
			end := strings.IndexByte(sql[i:], '\n')
			if end < 0 {
				i = len(sql)
			} else {
				i += end + 1
			}
			plain.WriteByte(' ')
		case strings.HasPrefix(sql[i:], "/*"):
			depth := 1
			i += 2
			for i < len(sql) && depth > 0 {
				if strings.HasPrefix(sql[i:], "/*") {
					depth++
					i += 2
				} else if strings.HasPrefix(sql[i:], "*/") {
					depth--
					i += 2
				} else {
					i++
				}
			}
			if depth != 0 {
				return fmt.Errorf("unterminated SQL comment")
			}
			plain.WriteByte(' ')
		case sql[i] == '\'' || sql[i] == '"':
			quote := sql[i]
			escape := quote == '\'' && i > 0 && (sql[i-1] == 'e' || sql[i-1] == 'E')
			i++
			closed := false
			for i < len(sql) {
				if escape && sql[i] == '\\' {
					i += 2
					continue
				}
				if sql[i] == quote {
					if i+1 < len(sql) && sql[i+1] == quote {
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return fmt.Errorf("unterminated SQL quote")
			}
			plain.WriteByte(' ')
		case sql[i] == '$':
			delimiter := dollarQuote.FindString(sql[i:])
			if delimiter == "" {
				plain.WriteByte(sql[i])
				i++
				continue
			}
			i += len(delimiter)
			end := strings.Index(sql[i:], delimiter)
			if end < 0 {
				return fmt.Errorf("unterminated SQL dollar quote")
			}
			i += end + len(delimiter)
			plain.WriteByte(' ')
		default:
			plain.WriteByte(sql[i])
			i++
		}
	}
	for _, statement := range strings.Split(plain.String(), ";") {
		words := strings.Fields(strings.ToUpper(statement))
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "BEGIN", "COMMIT", "END", "ROLLBACK", "ABORT", "START", "SAVEPOINT", "RELEASE":
			return fmt.Errorf("transaction control is not allowed; BaseStack owns the transaction")
		}
		if words[0] == "PREPARE" && len(words) > 1 && words[1] == "TRANSACTION" {
			return fmt.Errorf("prepared transactions are not allowed")
		}
	}
	return nil
}
