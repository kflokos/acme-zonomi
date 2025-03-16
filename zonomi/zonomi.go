// package zonomi contains a self-contained zonomi of a webhook that passes the cert-manager
// DNS conformance tests
package zonomi

import (
	"fmt"
	"os"
	"sync"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook"
	acme "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/miekg/dns"
	"k8s.io/client-go/rest"
)

type zonomiSolver struct {
	name       string
	server     *dns.Server
	txtRecords map[string]string
	sync.RWMutex
}

func (e *zonomiSolver) Name() string {
	return e.name
}

func (e *zonomiSolver) Present(ch *acme.ChallengeRequest) error {
    apiKey := os.Getenv("ZONOMI_API_KEY")
    if apiKey == "" {
        return fmt.Errorf("ZONOMI_API_KEY environment variable not set")
    }

    url := fmt.Sprintf("https://zonomi.com/app/dns/dyndns.jsp?action=SETTXT&name=%s&value=%s&apiKey=%s",
        ch.ResolvedFQDN, ch.Key, apiKey)

    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("failed to create TXT record: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("Zonomi API error: %s", string(body))
    }

    s.Log.Info("Successfully created TXT record", "fqdn", ch.ResolvedFQDN)
    return nil
}

func (e *zonomiSolver) CleanUp(ch *acme.ChallengeRequest) error {
    s.Log.Info("Deleting TXT record", "fqdn", ch.ResolvedFQDN)

    apiKey := os.Getenv("ZONOMI_API_KEY")
    if apiKey == "" {
        return fmt.Errorf("ZONOMI_API_KEY environment variable not set")
    }

    url := fmt.Sprintf("https://zonomi.com/app/dns/dyndns.jsp?action=REMOVETXT&name=%s&apiKey=%s",
        ch.ResolvedFQDN, apiKey)

    resp, err := http.Get(url)
    if err != nil {
        return fmt.Errorf("failed to delete TXT record: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("Zonomi API error: %s", string(body))
    }

    s.Log.Info("Successfully deleted TXT record", "fqdn", ch.ResolvedFQDN)
    return nil

}

func (e *zonomiSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	go func(done <-chan struct{}) {
		<-done
		if err := e.server.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
	}(stopCh)
	go func() {
		if err := e.server.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			os.Exit(1)
		}
	}()
	return nil
}

func New(port string) webhook.Solver {
	e := &zonomiSolver{
		name:       "zonomi",
		txtRecords: make(map[string]string),
	}
	e.server = &dns.Server{
		Addr:    ":" + port,
		Net:     "udp",
		Handler: dns.HandlerFunc(e.handleDNSRequest),
	}
	return e
}
