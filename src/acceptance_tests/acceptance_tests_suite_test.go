package acceptance_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	helpers_config "github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/onsi/gomega/gexec"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestAcceptanceTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AcceptanceTests Suite")
}

const Timeout_Short = 10 * time.Second

var (
	allDeployedInstances []instanceInfo
	config               *helpers_config.Config
)

var _ = BeforeSuite(func() {
	//By("deploying bosh-dns and bosh-dns-adapter", func() {
	//	cmd := exec.Command("bosh", "deploy", "-n", "-d", "acceptance", "./test_assets/manifest.yml")
	//	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	//	Expect(err).ToNot(HaveOccurred())
	//
	//	Eventually(session, 5*time.Minute).Should(gexec.Exit(0))
	//
	//	Expect(err).ToNot(HaveOccurred())
	//})

	//allDeployedInstances = getInstanceInfos()
	config = helpers_config.LoadConfig()

	Expect(cf.Cf("api", "--skip-ssl-validation", config.ApiEndpoint).Wait(Timeout_Short)).To(gexec.Exit(0))
	AuthAsAdmin()

	appsDir = os.Getenv("APPS_DIR")
	Expect(appsDir).NotTo(BeEmpty())
})

type instanceInfo struct {
	IP            string
	InstanceID    string
	InstanceGroup string
	Index         string
}

func AuthAsAdmin() {
	Auth(config.AdminUser, config.AdminPassword)
}

func Auth(username, password string) {
	By("authenticating as " + username)
	cmd := exec.Command("cf", "auth", username, password)
	sess, err := gexec.Start(cmd, nil, nil)
	Expect(err).NotTo(HaveOccurred())
	Eventually(sess.Wait(Timeout_Short)).Should(gexec.Exit(0))
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
