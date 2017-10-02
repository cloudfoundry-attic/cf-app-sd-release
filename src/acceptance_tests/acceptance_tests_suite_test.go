package acceptance_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"github.com/onsi/gomega/gexec"
	"os/exec"
	"time"
	"encoding/json"
	"strings"
)

func TestAcceptanceTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AcceptanceTests Suite")
}

var allDeployedInstances []instanceInfo

var _ = BeforeSuite(func() {
	By("deploying bosh-dns and bosh-dns-adapter", func() {
		cmd := exec.Command("bosh", "deploy", "-n", "-d", "acceptance", "./test_assets/manifest.yml")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session, 5*time.Minute).Should(gexec.Exit(0))

		Expect(err).ToNot(HaveOccurred())
	})

	allDeployedInstances = getInstanceInfos()
})

type instanceInfo struct {
	IP            string
	InstanceID    string
	InstanceGroup string
	Index         string
}

func getInstanceInfos() []instanceInfo {
	cmd := exec.Command("bosh", "-d", "acceptance", "instances", "--details", "--json")
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 20*time.Second).Should(gexec.Exit(0))

	var response struct {
		Tables []struct {
			Rows []map[string]string
		}
	}

	out := []instanceInfo{}

	json.Unmarshal(session.Out.Contents(), &response)

	for _, row := range response.Tables[0].Rows {
		instanceStrings := strings.Split(row["instance"], "/")

		out = append(out, instanceInfo{
			IP:            row["ips"],
			InstanceGroup: instanceStrings[0],
			InstanceID:    instanceStrings[1],
			Index:         row["index"],
		})
	}

	return out
}
