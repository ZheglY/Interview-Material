package domain

import "strings"

const maxEmailLength = 320

// Email — нормализованный адрес, который выступает идентификатором пользователя.
type Email struct {
	value string
}

// NewEmail проверяет и канонизирует email для поиска без учёта регистра.
func NewEmail(raw string) (Email, error) {
	// Удаляем только случайные пробелы вокруг значения, а не внутри адреса.
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" || len(normalized) > maxEmailLength {
		return Email{}, ErrInvalidEmail
	}

	localPart, domainPart, hasSeparator := strings.Cut(normalized, "@")
	if !hasSeparator || localPart == "" || domainPart == "" || strings.Contains(domainPart, "@") {
		return Email{}, ErrInvalidEmail
	}
	if len(localPart) > 64 || strings.HasPrefix(localPart, ".") ||
		strings.HasSuffix(localPart, ".") || strings.Contains(localPart, "..") {
		return Email{}, ErrInvalidEmail
	}

	// На первом этапе поддерживаем только ASCII email; EAI потребует отдельной IDNA-политики.
	if !isValidLocalPart(localPart) || !isValidDomain(domainPart) {
		return Email{}, ErrInvalidEmail
	}

	return Email{value: normalized}, nil
}

// Value возвращает нормализованный адрес для persistence и публичного ответа API.
func (email Email) Value() string {
	return email.value
}

// isValidLocalPart допускает распространённое безопасное подмножество ASCII email.
func isValidLocalPart(value string) bool {
	for index := 0; index < len(value); index++ {
		character := value[index]
		isLetter := character >= 'a' && character <= 'z'
		isDigit := character >= '0' && character <= '9'
		isAllowedSymbol := strings.ContainsRune(".!#$%&'*+-/=?^_`{|}~", rune(character))
		if !isLetter && !isDigit && !isAllowedSymbol {
			return false
		}
	}
	return true
}

// isValidDomain проверяет DNS-подобные ASCII-метки домена.
func isValidDomain(value string) bool {
	if len(value) > 253 || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}

	for _, label := range strings.Split(value, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for index := 0; index < len(label); index++ {
			character := label[index]
			isLetter := character >= 'a' && character <= 'z'
			isDigit := character >= '0' && character <= '9'
			if !isLetter && !isDigit && character != '-' {
				return false
			}
		}
	}
	return true
}