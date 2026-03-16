package helpers

import "golang.org/x/crypto/bcrypt"

func HashedPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	FatalError(err)

	return string(hashedPassword)
}

func CheckPassword(hashedPassword, inputPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))
	return err == nil
}
