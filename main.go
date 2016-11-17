package egfs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"time"

	"bytes"

	"github.com/gtank/cryptopasta"
)

type egfs struct {
	absolutePathToRepo string
	password           []byte
}

func (egfs egfs) Open(name string) (f http.File, err error) {
	d, err := egfs.Directory()
	if err != nil {
		return
	}
	for _, f := range d {
		if f.name == name {
			return f, nil
		}
	}
	return nil, errors.New("file not found")
}

func (egfs egfs) Directory() (files []*file, err error) {
	data, err := egfs.openAndDecryptFile("file")
	if err != nil {
		return
	}
	var fileNames map[string]interface{}
	json.Unmarshal(data, fileNames)
	for name := range fileNames {
		data, err := egfs.openAndDecryptFile(name)
		if err != nil {
			return nil, err
		}
		cmd := exec.Command("git", "log", "-1", "--format=%cd", "--date=iso8601")
		cmd.Dir = egfs.absolutePathToRepo
		jsonTime, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		var t time.Time
		json.Unmarshal(jsonTime, t)
		files = append(files, &file{
			content: bytes.NewBuffer(data),
			name:    name,
			modTime: t,
		})
	}
	return
}

func hashAndHex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func (egfs egfs) openAndDecryptFile(fileName string) (data []byte, err error) {
	var branch string
	if fileName == "" {
		branch = "master"
	} else {
		branch = hashAndHex(fileName)
	}
	cmd := exec.Command("git", "checkout", "-f", branch)
	cmd.Dir = egfs.absolutePathToRepo
	if err = cmd.Run(); err != nil {
		return
	}
	s, err := ioutil.ReadFile(filepath.Join(egfs.absolutePathToRepo, "file"))
	if err != nil {
		return
	}
	ciphertext, err := hex.DecodeString(string(s))
	if err != nil {
		return
	}
	key := sha256.Sum256(egfs.password)
	return cryptopasta.Decrypt(ciphertext, &key)
}

func (egfs egfs) writeAndEncryptFile(fileName string, fileContents []byte) (err error) {
	var branch string
	if fileName == "" {
		branch = "master"
	} else {
		branch = hashAndHex(fileName)
	}
	cmd := exec.Command("git", "checkout", "-f", branch)
	cmd.Dir = egfs.absolutePathToRepo
	if err = cmd.Run(); err != nil {
		return
	}
	key := sha256.Sum256(egfs.password)
	ciphertext, err := cryptopasta.Encrypt(fileContents, &key)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(filepath.Join(egfs.absolutePathToRepo, "file"), []byte(hex.EncodeToString(ciphertext)), 0644)
	if err != nil {
		return
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = egfs.absolutePathToRepo
	if err = cmd.Run(); err != nil {
		return
	}
	cmd = exec.Command("git", "commit", "-m", `"update"`)
	cmd.Dir = egfs.absolutePathToRepo
	if err = cmd.Run(); err != nil {
		return
	}
	if branch != "master" {
		data, err := egfs.openAndDecryptFile("")
		if err != nil {
			return err
		}
		var fileNames map[string]interface{}
		json.Unmarshal(data, fileNames)
		fileNames[fileName] = true
		b, _ := json.Marshal(fileNames)
		egfs.writeAndEncryptFile("", b)
	}
	return
}
