package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/crypto/pkcs12"
)

var TOKEN_ID string
var P12_PASSWORD string
var API_URL string

type startcomRequest struct {
	TokenID    string
	ActionType string
	CertType   string
	Domains    string
	CSR        string
}

type startcomResponseData struct {
	OrderId                         string
	OrderNo                         string
	OrderStatus                     int
	Certificate                     string
	CertificateFieldMD5             string
	IntermediateCertificate         string
	IntermediateCertificateFieldMD5 string
}

type startcomResponse struct {
	Status    int
	ErrorCode int
	ShortMsg  string
	Data      startcomResponseData
}

func NewStartcomResponse(body []byte) *startcomResponse {
	var r startcomResponse
	json.Unmarshal(body, &r)

	c, err := base64.StdEncoding.DecodeString(r.Data.Certificate)
	if err != nil {
		log.Fatal("cert problem")
	}

	i, err := base64.StdEncoding.DecodeString(r.Data.IntermediateCertificate)
	if err != nil {
		log.Fatal("intermediate cert problem")
	}

	r.Data.Certificate = string(c)
	r.Data.IntermediateCertificate = string(i)
	return &r
}

func loadClientCert(filename, password string) tls.Certificate {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal("ERROR READING FILE")
	}
	data, err := ioutil.ReadAll(f)

	blocks, err := pkcs12.ToPEM(data, password)
	c := make(map[int][]byte)
	for n, b := range blocks {
		c[n] = pem.EncodeToMemory(b)
	}
	cert, err := tls.X509KeyPair(c[0], c[2])
	if err != nil {
		log.Fatal(err)
	}
	return cert
}

func buildCertificateRequest() *x509.CertificateRequest {
	template := &x509.CertificateRequest{
		//Signature: []byte("hi"),
		Subject: pkix.Name{
			Country:      []string{""},
			Organization: []string{""},
		},
		DNSNames:       []string{""},
		EmailAddresses: []string{""},
	}
	//fmt.Println(template)
	return template
}

func buildRequestForm(domains []string, csr []byte, vtype string) url.Values {
	req := startcomRequest{TOKEN_ID, "ApplyCertificate", vtype, domains[0], string(csr)}
	mehasdf, _ := json.Marshal(req)
	form := url.Values{}
	form.Add("RequestData", string(mehasdf))
	return form
}

func generateCsrAndKey(keybits int) ([]byte, []byte) {
	template := buildCertificateRequest()
	//template :=

	privatekey, err := rsa.GenerateKey(rand.Reader, keybits)
	if err != nil {
		fmt.Println(err)
	}
	k := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privatekey)})

	csr, err := x509.CreateCertificateRequest(rand.Reader, template, privatekey)
	if err != nil {
		fmt.Println(err)
	}
	c := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})

	return c, k
}

func doRequest(cert tls.Certificate, postForm url.Values) *startcomResponse {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}
	resp, err := client.PostForm(API_URL, postForm)
	if err != nil {
		log.Fatal(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)

	//fmt.Println(string(body))
	r := NewStartcomResponse(body)
	//fmt.Println(r.Data.Certificate)
	return r
}

func writeCertsAndKey(domain, cert, i_cert, key string) {
	c := []byte(cert)
	i := []byte(i_cert)
	k := []byte(key)

	err := ioutil.WriteFile(domain+".key", k, 0644)
	if err != nil {
		log.Fatal("Failed to write key file")
	}
	err = ioutil.WriteFile(domain+"-intermediate.crt", i, 0644)
	if err != nil {
		log.Fatal("Failed to write Intermediate Cert file")
	}

	err = ioutil.WriteFile(domain+".crt", c, 0644)
	if err != nil {
		log.Fatal("Failed to write Cert file")
	}

	err = ioutil.WriteFile(domain+"-chained.crt", append(c, i...), 0644)
	if err != nil {
		log.Fatal("Failed to write Chained Cert file")
	}

}

func main() {
	P12_PASSWORD := os.Getenv("STARTCOM_API_CERT_PASSWORD")
	if P12_PASSWORD == "" {
		log.Fatal("Please set the STARTCOM_API_CERT_PASSWORD environment variable")
	}
	TOKEN_ID = os.Getenv("STARTCOM_API_TOKEN_ID")
	if TOKEN_ID == "" {
		log.Fatal("Please set the STARTCOM_API_TOKEN_ID environment variable")
	}

	testF := flag.Bool("test", false, "test mode (generates 1 day certs against apitest)")
	typeF := flag.String("type", "dvssl", "type of cert to generate, default dvssl. options: ovssl evssl ivssl")
	keyB := flag.Int("b", 2048, "how many bits for the rsa key? 2048, 4096, more??")
	domainsPtr := flag.String("d", "", "domain list")
	flag.Parse()

	if *testF {
		API_URL = "https://apitest.startssl.com"
	} else {
		API_URL = "https://api.startssl.com"
	}
	if *domainsPtr == "" {
		log.Fatal("You must specify at least one domain with `-d [domain]`")
	}

	fmt.Println("Generating Key and Certificate for domains: ", *domainsPtr)

	//This is the client cert issued by startcom to access the api
	cert := loadClientCert("cert.p12", P12_PASSWORD)

	//We Generate a Csr that is mostly useless and a Key
	csr, key := generateCsrAndKey(*keyB)

	//fmt.Println(string(key))
	res := doRequest(cert, buildRequestForm([]string{*domainsPtr}, csr, *typeF))
	if res.Status == 1 {
		fmt.Printf("Successfully generated cert and key for: %s\n", *domainsPtr)
		writeCertsAndKey(*domainsPtr, res.Data.Certificate, res.Data.IntermediateCertificate, string(key))
	} else {
		fmt.Printf("Failed to generate cert and key for: %s\n", *domainsPtr)
		fmt.Printf("Startcom returned the error: %s\n", res.ShortMsg)
	}

}
