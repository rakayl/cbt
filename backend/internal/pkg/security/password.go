package security

import "golang.org/x/crypto/bcrypt"

func HashPassword(password, pepper string, cost int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password+pepper), cost)
	return string(b), err
}
func CheckPassword(hash, password, pepper string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+pepper)) == nil
}
