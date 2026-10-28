package setting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKefuCreationOffersProfessionalAvatars(t *testing.T) {
	templatePath := filepath.Join("..", "static", "templates", "default", "setting_kefu_list.html")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read kefu template: %v", err)
	}
	source := string(content)
	for _, family := range []string{"professional", "chinese"} {
		for _, gender := range []string{"woman", "man"} {
			for index := 1; index <= 5; index++ {
				name := fmt.Sprintf("%s-%s-%02d.jpg", family, gender, index)
				if !strings.Contains(source, name) {
					t.Errorf("creation form does not offer %s", name)
				}
				imagePath := filepath.Join("..", "static", "images", "support-avatars", name)
				info, err := os.Stat(imagePath)
				if err != nil {
					t.Errorf("support avatar %s is unavailable: %v", name, err)
					continue
				}
				if info.Size() >= 1024*1024 {
					t.Errorf("support avatar %s is %d bytes; want less than 1 MB", name, info.Size())
				}
			}
		}
	}
}
