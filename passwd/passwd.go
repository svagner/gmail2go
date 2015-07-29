// Encrypter/decrypter for accounts and password file
package passwd

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/howeyc/gopass"
	"github.com/svagner/gmail2go/config"
	"github.com/svagner/gmail2go/logger"
)

type AccountAuth struct {
	UserName string
	Password string
}

type Passwd map[string]AccountAuth

// Encrypts data to destrination writer
func encrypt(dst io.Writer, data *bytes.Buffer, key, iv []byte) (err error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	w := &cipher.StreamWriter{S: cipher.NewOFB(c, iv), W: dst}
	io.Copy(w, data)
	return
}

// Returns decrypted data from src reader
func decrypt(src io.Reader, key, iv []byte) (data *bytes.Buffer, err error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return
	}
	r := &cipher.StreamReader{S: cipher.NewOFB(c, iv), R: src}
	data = new(bytes.Buffer)
	io.Copy(data, r)
	return
}

func NewPasswd(file string) (Passwd, error) {
	pf := make(Passwd)
	fin, err := os.Open(file)
	if err == nil {
		// if the config file was opened decrypt it and unmarshal accounts map
		r := bufio.NewReader(fin)
		res, err := decrypt(r, make([]byte, 16), make([]byte, 16))
		if err != nil {
			return nil, err
		} else {
			dec := json.NewDecoder(res)
			if err := dec.Decode(&pf); err != nil {
				logger.CriticalPrint("Could not decode accounts content to json: ", err)
			}
			if len(pf) == 0 {
				pf = make(Passwd)
			}
		}
	} else {
		fin, err = os.Create(file)
		if err != nil {
			return nil, err
		}
		defer fin.Close()
		w := bufio.NewWriter(fin)
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		err := enc.Encode(pf)
		if err != nil {
			logger.CriticalPrint("Could not encode accounts to json", err)
		} else {
			err = encrypt(w, &out, make([]byte, 16), make([]byte, 16))
			if err != nil {
				logger.CriticalPrint("Could not encrypt accounts json string to file", err)
			}
			w.Flush()
		}
	}
	return pf, nil
}

func (pf *Passwd) Check(file string, accounts map[string]*config.AccountConfig) {
	check := false
	for key, value := range accounts {
		if _, ok := (*pf)[key]; !ok {
			fmt.Printf("Password for profile \"%s\" account %s: ", key, value.Email)
			(*pf)[key] = AccountAuth{UserName: value.Email, Password: string(gopass.GetPasswd())}
			check = true
		}
	}
	if check {
		fin, err := os.Create(file)
		if err != nil {
			log.Fatalln(err)
		}
		defer fin.Close()
		w := bufio.NewWriter(fin)
		var out bytes.Buffer
		enc := json.NewEncoder(&out)
		err = enc.Encode(*pf)
		if err != nil {
			logger.CriticalPrint("Could not encode accounts to json", err)
		} else {
			err = encrypt(w, &out, make([]byte, 16), make([]byte, 16))
			if err != nil {
				logger.CriticalPrint("Could not encrypt accounts json string to file", err)
			}
			w.Flush()
		}
	}
}
