package egfs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gtank/cryptopasta"
)

type egfs struct {
	absolutePathToRepo string
	password           []byte
}

func (egfs egfs) addEntry(documentName string, entry entry) (err error) {
	// step 1. switch to the branch containing entries for this document
	if err = egfs.run("git", "checkout", "-b", hashAndHex(documentName)); err != nil {
		return
	}
	// step 3. write the entry to a file whose name is a timestamp
	// TODO use UUIDv1
	n := filepath.Join(egfs.absolutePathToRepo, newTimestampUUID(entry.timestamp))
	if err = encryptAndWrite(n, egfs.password, entry.content); err != nil {
		return
	}
	// step 4. add-commit-push the new file
	if err = egfs.run("git", "add", "."); err != nil {
		return
	}
	if err = egfs.run("git", "commit", "-m", "update"); err != nil {
		return
	}
	if err = egfs.run("git", "push"); err != nil {
		return
	}
	// step 5. see if this document is listed in the main file, if not then add it
	docs, err := egfs.listDocuments()
	for _, doc := range docs {
		if doc.name == documentName {
			return nil
		}
	}
	// step 5b. go to master and get document list
	if err = egfs.run("git", "checkout", "-f", "master"); err != nil {
		return
	}
	var documentNames map[string]interface{}
	data, err := openAndDecrypt(filepath.Join(egfs.absolutePathToRepo, ".table-of-contents"), egfs.password)
	if err != nil {
		return
	}
	json.Unmarshal(data, &documentNames)
	// step 5c. add this doucment to the master list
	documentNames[documentName] = true // TODO add meta data?
	data, _ = json.Marshal(documentNames)
	encryptAndWrite(filepath.Join(egfs.absolutePathToRepo, ".table-of-contents"), egfs.password, data)
	if err = egfs.run("git", "add", "."); err != nil {
		return
	}
	if err = egfs.run("git", "commit", "-m", `"update"`); err != nil {
		return
	}
	return egfs.run("git", "push")
}

type document struct {
	name    string
	entries []entry
}

type entry struct {
	timestamp time.Time
	content   []byte
}

func (egfs egfs) listDocuments() (documents []document, err error) {
	// step 1. go to master branch
	if err = egfs.run("git", "checkout", "-f", "master"); err != nil {
		return
	}
	// step 2. read encrypted list of document names
	var documentNames map[string]interface{}
	data, err := openAndDecrypt(filepath.Join(egfs.absolutePathToRepo, ".table-of-contents"), egfs.password)
	if err != nil {
		return
	}
	json.Unmarshal(data, &documentNames)
	// step 3. for each document read all entries
	for name := range documentNames {
		document := document{name: name}
		if err = egfs.run("git", "checkout", "-f", hashAndHex(name)); err != nil {
			return
		}
		filepath.Walk(egfs.absolutePathToRepo, func(path string, info os.FileInfo, _ error) error {
			if info.IsDir() {
				return nil
			}
			t := parseTimestampUUID(info.Name())
			if t == nil {
				return nil
			}
			b, err := openAndDecrypt(path, egfs.password)
			if err != nil {
				return err
			}
			document.entries = append(document.entries, entry{timestamp: *t, content: b})
			return nil
		})
	}
	return
}

// utility functions

func (egfs egfs) run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = egfs.absolutePathToRepo
	return cmd.Run()
}

func hashAndHex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func openAndDecrypt(fileName string, password []byte) ([]byte, error) {
	s, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256(password)
	ciphertext, err := hex.DecodeString(string(s))
	if err != nil {
		return nil, err
	}
	return cryptopasta.Decrypt(ciphertext, &key)
}

func encryptAndWrite(fileName string, password, content []byte) (err error) {
	key := sha256.Sum256(password)
	ciphertext, err := cryptopasta.Encrypt(content, &key)
	if err != nil {
		return
	}
	return ioutil.WriteFile(fileName, []byte(hex.EncodeToString(ciphertext)), 0644)
}

func newTimestampUUID(t time.Time) string {
	b, _ := t.MarshalBinary()
	return hex.EncodeToString(b)
}

func parseTimestampUUID(s string) *time.Time {
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil
	}
	t := new(time.Time)
	if err = t.UnmarshalBinary(b); err != nil {
		return nil
	}
	return t
}
