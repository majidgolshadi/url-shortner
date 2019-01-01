package url_shortner

import (
	"testing"
)

func getTokenGeneratorTest() *tokenGenerator {
	counter := &MockCounter{
		Offset: 1,
	}

	datastore := &MockDataStore{}

	return NewTokenGenerator(counter, datastore)
}

func TestNewUrl(t *testing.T) {
	tg := getTokenGeneratorTest()
	token := tg.NewUrl("farmx.ir")

	if token != "2" {
		t.Fail()
	}
}

func TestPredefinedNewUrl(t *testing.T) {
	tg := getTokenGeneratorTest()
	token := tg.NewUrl("http://test.domain.com")

	if token != "testToken" {
		t.Fail()
	}
}

func TestNewUrlWithCustomToken(t *testing.T) {
	tg := getTokenGeneratorTest()
	token, err := tg.NewUrlWithCustomToken("farmx.ir", "custom_token")
	if err != nil {
		t.Fail()
	}

	if token != "custom_token" {
		t.Fail()
	}
}

func TestPredefinedUrlWithCustomToken(t *testing.T) {
	tg := getTokenGeneratorTest()
	token, err := tg.NewUrlWithCustomToken("http://test.domain.com", "sample_token")
	if err == nil {
		t.Fail()
	}

	if token == "sample_token" {
		t.Fail()
	}

	if token != "testToken" {
		t.Fail()
	}
}

func TestNewUrlWithCustomUsedToken(t *testing.T) {
	tg := getTokenGeneratorTest()
	token, err := tg.NewUrlWithCustomToken("farmx.ir", "usedToken")

	if err == nil {
		t.Fail()
	}

	if token != "" {
		t.Fail()
	}
}

func TestGetLongUrl(t *testing.T) {
	tg := getTokenGeneratorTest()
	if url := tg.GetLongUrl("testToken"); url != "http://test.domain.com" {
		t.Fail()
	}
}
