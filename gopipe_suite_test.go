package gopipe_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGopipeline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gopipe Suite")
}
