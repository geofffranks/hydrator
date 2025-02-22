package hydrate_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/hydrator/imagefetcher"
	testhelpers "code.cloudfoundry.org/hydrator/testhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestHydrate(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(time.Second * 300)
	RunSpecs(t, "Hydrate Suite")
}

var (
	ociImagePath string
	keep         bool
	helpers      *testhelpers.Helpers
)

var _ = SynchronizedBeforeSuite(func() []byte {
	var wincBin, grootBin, grootImageStore, diffBin, beforeSuiteOciImagePath string

	hydrateBin, err := gexec.Build("code.cloudfoundry.org/hydrator/cmd/hydrate")
	Expect(err).NotTo(HaveOccurred())

	if runtime.GOOS == "windows" {
		beforeSuiteOciImagePath, keep = os.LookupEnv("OCI_IMAGE_PATH")

		var present bool

		if !keep {
			var err error
			beforeSuiteOciImagePath, err = ioutil.TempDir("", "oci-image-path")

			logger := log.New(os.Stdout, "", 0)

			imageName, present := os.LookupEnv("IMAGE_NAME")
			Expect(present).To(BeTrue(), "IMAGE_NAME not set")

			imageTag, present := os.LookupEnv("IMAGE_TAG")
			Expect(present).To(BeTrue(), "IMAGE_TAG not set")

			imagefetcher.New(logger, beforeSuiteOciImagePath, imageName, imageTag, "", true).Run()
			Expect(err).ToNot(HaveOccurred())
		}

		grootBin, present = os.LookupEnv("GROOT_BINARY")
		Expect(present).To(BeTrue(), "GROOT_BINARY not set")

		grootImageStore, present = os.LookupEnv("GROOT_IMAGE_STORE")
		Expect(present).To(BeTrue(), "GROOT_IMAGE_STORE not set")

		wincBin, present = os.LookupEnv("WINC_BINARY")
		Expect(present).To(BeTrue(), "WINC_BINARY not set")

		diffBin, present = os.LookupEnv("DIFF_EXPORTER_BINARY")
		Expect(present).To(BeTrue(), "DIFF_EXPORTER_BINARY not set")

		// create a temporary test helpers object so we can "groot pull" on one node only
		rootfsURI := fmt.Sprintf("oci:///%s", filepath.ToSlash(beforeSuiteOciImagePath))
		testhelpers.NewHelpers(wincBin, grootBin, grootImageStore, diffBin, hydrateBin, true).GrootPull(rootfsURI)
	}

	return []byte(fmt.Sprintf("%s^%s^%s^%s^%s^%s",
		wincBin, grootBin, grootImageStore, diffBin, hydrateBin, beforeSuiteOciImagePath))

}, func(data []byte) {
	helperArgs := strings.Split(string(data), "^")
	wincBin := helperArgs[0]
	grootBin := helperArgs[1]
	grootImageStore := helperArgs[2]
	diffBin := helperArgs[3]
	hydrateBin := helperArgs[4]
	ociImagePath = helperArgs[5]

	debug, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	helpers = testhelpers.NewHelpers(wincBin, grootBin, grootImageStore, diffBin, hydrateBin, debug)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()

	if !keep {
		Expect(os.RemoveAll(ociImagePath)).To(Succeed())
	}
})
