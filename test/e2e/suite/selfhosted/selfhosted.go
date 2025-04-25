package selfhosted

import (
	"bytes"
	"encoding/json"
	"log"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/itchyny/gojq"
)

var _ = BeforeSuite(func() {
	cmd := exec.Command("helm", "repo", "add", "twuni", "https://helm.twun.io")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	Expect(cmd.Run()).NotTo(HaveOccurred())
	cmd = exec.Command("helm", "upgrade", "--install", "--create-namespace", "-n", "registry", "--wait", "--set", "service.type=NodePort", "--set", "service.nodePort=30443", "-f", "./manifests/docker-registry-values.yaml", "docker-registry", "twuni/docker-registry")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	Expect(cmd.Run()).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	cmd := exec.Command("helm", "uninstall", "-n", "registry", "--wait", "docker-registry")
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	Expect(cmd.Run()).NotTo(HaveOccurred())
})

var _ = Describe("version-checker selfhosted", func() {
	BeforeEach(func() {
		cmd := exec.Command("kubectl", "apply", "-f", "./manifests/kaniko.yaml", "-f", "./manifests/docker-credentials.yaml")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).NotTo(HaveOccurred())
		cmd = exec.Command("kubectl", "wait", "pod", "-lapp=e2e-kaniko", "--timeout=30s", "--for=jsonpath={.status.containerStatuses[*].state.terminated.reason}=Completed")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		cmd := exec.Command("kubectl", "delete", "-f", "./manifests/kaniko.yaml")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		cmd := exec.Command("kubectl", "apply", "-f", "./manifests/prom2json.yaml")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).NotTo(HaveOccurred())
		cmd = exec.Command("kubectl", "wait", "--for=condition=Complete", "--timeout=30s", "job", "-ljob-name=prom2json")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		cmd := exec.Command("kubectl", "delete", "-f", "./manifests/prom2json.yaml")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).NotTo(HaveOccurred())
	})

	When("a Pod is deployed", func() {
		BeforeEach(func() {
			cmd := exec.Command("kubectl", "apply", "-f", "./manifests/image-from-reg.yaml")
			cmd.Stdout = GinkgoWriter
			cmd.Stderr = GinkgoWriter
			Expect(cmd.Run()).NotTo(HaveOccurred())

		})
		AfterEach(func() {
			cmd := exec.Command("kubectl", "delete", "-f", "./manifests/image-from-reg.yaml")
			cmd.Stdout = GinkgoWriter
			cmd.Stderr = GinkgoWriter
			Expect(cmd.Run()).NotTo(HaveOccurred())
		})

		It("it should get the current version", func() {
			buf := new(bytes.Buffer)
			cmd := exec.Command("kubectl", "logs", "-ljob-name=prom2json")
			cmd.Stdout = buf
			cmd.Stderr = GinkgoWriter
			Expect(cmd.Run()).NotTo(HaveOccurred())

			//k logs -ljob-name=prom2json | jq '.[]|select(.name=="version_checker_is_latest_version")| .metrics[] | select(.labels.image=="docker-registry.registry.svc.cluster.local:5000/my-app") | .labels.current_version'
			//k logs -ljob-name=prom2json | jq '.[]|select(.name=="version_checker_is_latest_version")| .metrics[] | select(.labels.image=="docker-registry.registry.svc.cluster.local:5000/my-app") | .labels.latest_version'
			//k logs -ljob-name=prom2json | jq '.[]|select(.name=="version_checker_is_latest_version")| .metrics[] | select(.labels.image=="docker-registry.registry.svc.cluster.local:5000/my-app") | .value'
			query, err := gojq.Parse(".[]|select(.name==\"version_checker_is_latest_version\")| .metrics[] | select(.labels.image==\"docker-registry.registry.svc.cluster.local:5000/my-app\") | .labels.current_version")
			if err != nil {
				log.Fatalln(err)
			}
			var result []interface{}
			err = json.Unmarshal(buf.Bytes(), &result)
			if err != nil {
				log.Fatalln(err)
			}
			iter := query.Run(result)
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
						break
					}
					log.Fatalln(err)
				}
				Expect(v).To(Equal("0.0.1"))
			}
		})

		It("it should find a newer version", func() {
			buf := new(bytes.Buffer)
			cmd := exec.Command("kubectl", "logs", "-ljob-name=prom2json")
			cmd.Stdout = buf
			cmd.Stderr = GinkgoWriter
			Expect(cmd.Run()).NotTo(HaveOccurred())

			//k logs -ljob-name=prom2json | jq '.[]|select(.name=="version_checker_is_latest_version")| .metrics[] | select(.labels.image=="docker-registry.registry.svc.cluster.local:5000/my-app") | .labels.current_version'
			//k logs -ljob-name=prom2json | jq '.[]|select(.name=="version_checker_is_latest_version")| .metrics[] | select(.labels.image=="docker-registry.registry.svc.cluster.local:5000/my-app") | .labels.latest_version'
			//k logs -ljob-name=prom2json | jq '.[]|select(.name=="version_checker_is_latest_version")| .metrics[] | select(.labels.image=="docker-registry.registry.svc.cluster.local:5000/my-app") | .value'
			query, err := gojq.Parse(".[]|select(.name==\"version_checker_is_latest_version\")| .metrics[] | select(.labels.image==\"docker-registry.registry.svc.cluster.local:5000/my-app\") | .labels.latest_version")
			if err != nil {
				log.Fatalln(err)
			}
			var result []interface{}
			err = json.Unmarshal(buf.Bytes(), &result)
			if err != nil {
				log.Fatalln(err)
			}
			iter := query.Run(result)
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
						break
					}
					log.Fatalln(err)
				}
				Expect(v).To(Equal("0.0.2"))
			}
		})
	})
})
