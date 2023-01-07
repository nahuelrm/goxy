package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func main() {

	// options setup
	var concurrency int
	flag.IntVar(&concurrency, "c", 20, "set the concurrency level.")

	var domain string
	flag.StringVar(&domain, "d", "", "set the domain to start the scan.")

	var complete bool
	flag.BoolVar(&complete, "complete", true, "set the complete scan mode (default scan mode).")

	var keyword string
	flag.StringVar(&keyword, "keyword", "", "set the keyword to search for domains that contain it.")

	flag.Parse()

	// initial validations
	if len(domain) == 0 {
		fmt.Println("\033[31m[!] You must specify a domain to use this tool.\033[0m")
		os.Exit(2)
	}

	validateDomain(domain)

	if len(keyword) > 0 {
		complete = false
	}

	// get company name to validate that every root domain obtained belongs to the same
	var company string = getCompanyName(domain)

	if len(strings.TrimSpace(company)) == 0 {
		fmt.Println("\033[31m[!] Could not find company name in https://whoxy.sh\033[0m")
		os.Exit(2)
	}

	// output program setup
	outputCH := make(chan string)
	uniqueDomains := sync.Map{}
	var outputWG sync.WaitGroup
	outputWG.Add(1)
	go func() {
		defer outputWG.Done()
		for o := range outputCH {
			if _, ok := uniqueDomains.Load(o); !ok {
				fmt.Println(o)
				uniqueDomains.Store(o, true)
			}
		}
	}()

	// scan mode
	if complete {
		completeScan(domain, company, concurrency, outputCH)
	} else {
		keywordScan(keyword, company, concurrency, outputCH)
	}

	outputWG.Wait()
}

func completeScan(domain, company string, concurrency int, outputCH chan string) {
	companyId := getCompanyId(domain)

	cmdCompany := "curl -s https://www.whoxy.com/" + companyId + " | htmlq 'td[class=\"left nowrap\"]' | htmlq 'a[href]' | grep -oP '(?<=href=\")[^\"]*' | sed 's|../||' | httprobe -c 80 | sed 's|https://||' | sed 's|http://||'"

	outCompany, errCompany := exec.Command("bash", "-c", cmdCompany).Output()
	if errCompany != nil {
		fmt.Fprintf(os.Stderr, "Failed to run command: %s\n", errCompany)
		os.Exit(2)
	}

	scannerCompany := bufio.NewScanner(strings.NewReader(string(outCompany)))
	scannerCompany.Split(bufio.ScanLines)
	for scannerCompany.Scan() {
		outputCH <- scannerCompany.Text()
	}

	emailId := getEmailId(domain)

	cmdEmail := "curl -s https://www.whoxy.com/" + emailId + " | htmlq 'td[class=\"left nowrap\"]' | htmlq 'a[href]' | grep -oP '(?<=href=\")[^\"]*' | sed 's|../||' | httprobe -c 80 | sed 's|https://||' | sed 's|http://||'"

	outEmail, errEmail := exec.Command("bash", "-c", cmdEmail).Output()
	if errEmail != nil {
		fmt.Fprintf(os.Stderr, "Failed to run command: %s\n", errEmail)
		fmt.Println(cmdEmail)
		os.Exit(2)
	}

	scannerEmail := bufio.NewScanner(strings.NewReader(string(outEmail)))
	scannerEmail.Split(bufio.ScanLines)
	for scannerEmail.Scan() {
		outputCH <- scannerEmail.Text()
	}

	keywordScan(strings.Split(domain, ".")[0], company, concurrency, outputCH)
}

func keywordScan(keyword, company string, concurrency int, outputCH chan string) {
	cmd := "curl -s https://www.whoxy.com/keyword/" + keyword + " | htmlq 'td[class=\"left nowrap\"]' | htmlq 'a[href]' | grep -oP '(?<=href=\")[^\"]*' | sed 's|../||' | httprobe -c 80 | sed 's|https://||' | sed 's|http://||' | sort -u"

	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run command: %s\n", err)
		os.Exit(2)
	}

	whoisCH := make(chan string)
	var whoisWG sync.WaitGroup
	whoisWG.Add(concurrency/2)
	for i := 0; i < concurrency/2; i++ {
		go func() {
			defer whoisWG.Done()

			for domain := range whoisCH {
				whoisDomain(domain, company, outputCH)
			}
		}()
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		whoisCH <- scanner.Text()
	}

	close(whoisCH)
	whoisWG.Wait()
	close(outputCH)
}

func whoisDomain(domain, company string, outputCH chan string) {
	cmd := "whois " + domain

	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil { //TODO check if statement
		return
	}

	if strings.Contains(string(out), company) {
		outputCH <- domain
	}
}

func getCompanyName(domain string) string {
	cmd := "curl -s https://www.whoxy.com/" + domain + " | htmlq | grep '\"company_name\":' | head -n 1 | sed 's/.*company_name\": //' | sed 's/\"//g' | sed 's/,//' | xargs"

	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil { //TODO: check if statement
		fmt.Fprintf(os.Stderr, "Failed to run command: %s\n", err)
		os.Exit(2)
	}
	return string(out)
}

func getCompanyId(domain string) string {
	cmd := "curl -s https://www.whoxy.com/" + domain + " | htmlq 'a[href]' | grep \"company/\" | grep -oP '(?<=href=\")[^\"]*' | sort -u"

	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run command: %s\n", err)
		os.Exit(2)
	}
	return strings.TrimSpace(string(out))
}

func getEmailId(domain string) string {
	cmd := "curl -s https://www.whoxy.com/" + domain + " | htmlq 'a[href]' | grep \"email/\" | grep -oP '(?<=href=\")[^\"]*' | sort -u"

	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run command: %s\n", err)
		os.Exit(2)
	}
	return strings.TrimSpace(string(out))
}

func validateDomain(domain string) {
	if strings.Contains(domain, "https://") || strings.Contains(domain, "http://") {
		fmt.Println("\033[31m[!] Your domain should not contain http:// or https://.\033[0m")
		os.Exit(2)
	}
}
