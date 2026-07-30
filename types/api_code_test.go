package types

import (
	"sync"
	"testing"
)

func TestApiCode(t *testing.T) {
	t.Log(ApiCode.GetMessage(ApiCode.SUCCESS))
}

func TestApiCodeConcurrentLanguageAccess(t *testing.T) {
	var wait sync.WaitGroup
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			if index%2 == 0 {
				ApiCode.SetLanguage("cn")
			} else {
				ApiCode.SetLanguage("en")
			}
			_ = ApiCode.GetMessage(ApiCode.SUCCESS)
		}(index)
	}
	wait.Wait()
	ApiCode.SetLanguage("cn")
}
