package services

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"zyra-api/internal/models"
)

var labels = map[string]string{
	"Datafiles": "datafiles", "Backups": "backups", "Tablespaces": "tablespace", "Segments": "segments", "Extents": "extents", "FailedJobs": "failed_jobs", "Indexes": "indexes", "InvalidObjects": "invalid_objects", "ArchiveDestinations": "archive_destinations", "RecoveryAreaSpace": "fra", "ASM_Space": "asm_space", "Clusterware": "clusterware", "MaxLag": "max_lag",
}
var sectionRE = regexp.MustCompile(`^([A-Za-z_ ]+)=(.*)$`)
var freeRE = regexp.MustCompile(`^([^\s]+)\s+.*?([0-9]+(?:\.[0-9]+)?)%\s*$`)
var filesystemRE = regexp.MustCompile(`^([^\s]+)\s+([0-9]+(?:\.[0-9]+)?)%\s+-`)
var linuxFilesystemRE = regexp.MustCompile(`^\S+\s+(\S+)\s+([0-9]+(?:\.[0-9]+)?)%$`)
var scriptRE = regexp.MustCompile(`^Script\s*:\s*\S+`)
var reportRE = regexp.MustCompile(`(?m)^ReportOn:\s*([^,\r\n]+)`)
var uniqueIDRE = regexp.MustCompile(`(?m)^UniqueID:[ \t]*([^\r\n]*)`)
var databaseNameRE = regexp.MustCompile(`(?m)^Database:\s*([^,\r\n]+)`)
var hostRE = regexp.MustCompile(`(?m)^Run by\s*:\s*[^\r\n]*@([^\s\r\n]+)`)

func knownCheck(k string) bool {
	if k == "filesystem" || k == "missing_email" {
		return true
	}
	for _, v := range labels {
		if k == v {
			return true
		}
	}
	return false
}
func ValidateSettings(v models.CheckSettings) error {
	for k := range v.Selected {
		if !knownCheck(k) {
			return invalid("unsupported check: " + k)
		}
	}
	for k, resources := range v.Resources {
		if !knownCheck(k) {
			return invalid("unsupported check: " + k)
		}
		if len(resources) > 0 && k != "filesystem" && k != "tablespace" && k != "asm_space" && k != "backups" {
			return invalid("resource rules are not supported for check: " + k)
		}
		for name, r := range resources {
			if strings.TrimSpace(name) == "" {
				return invalid("resource name cannot be empty")
			}
			if (k == "filesystem" && r.MinFreePercent != nil) || ((k == "tablespace" || k == "asm_space") && r.MaxUsedPercent != nil) || (k == "backups" && (r.MinFreePercent != nil || r.MaxUsedPercent != nil)) {
				return invalid("unsupported threshold for check: " + k)
			}
			for _, p := range []*float64{r.MinFreePercent, r.MaxUsedPercent} {
				if p != nil && (math.IsNaN(*p) || math.IsInf(*p, 0) || *p < 0 || *p > 100) {
					return invalid("threshold must be between 0 and 100")
				}
			}
		}
	}
	return nil
}

type Parsed struct {
	UniqueID             string
	ReportTime, Hostname string
	Complete             bool
	Results              []models.CheckResult
}

func ParseReport(body string, settings models.CheckSettings) Parsed {
	p := Parsed{Results: []models.CheckResult{}}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	if m := uniqueIDRE.FindStringSubmatch(body); len(m) > 1 {
		p.UniqueID = strings.TrimSpace(m[1])
	}
	if m := reportRE.FindStringSubmatch(body); len(m) > 1 {
		p.ReportTime = strings.TrimSpace(m[1])
	}
	if m := hostRE.FindStringSubmatch(body); len(m) > 1 {
		p.Hostname = strings.TrimSpace(m[1])
	}
	sections := map[string]string{}
	current := ""
	linuxFilesystem := false
	scriptFooter := false
	scriptHeading := false
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		// Only strip a trailing report continuation marker. Interior path separators remain intact.
		line = strings.TrimSpace(strings.TrimSuffix(line, `\`))
		if line == "Script Info" {
			scriptHeading = true
		}
		if scriptRE.MatchString(strings.Join(strings.Fields(line), " ")) {
			scriptFooter = true
			current = ""
			continue
		}
		if line == "Database Stats" || line == "Script Info" {
			current = ""
			continue
		}
		if line == "Filesystem Usage" || strings.Join(strings.Fields(line), " ") == "Filesystem Mounted Use%" {
			linuxFilesystem = line != "Filesystem Usage"
			current = "filesystem"
			sections[current] = ""
			continue
		}
		if m := sectionRE.FindStringSubmatch(line); len(m) > 0 {
			if key, ok := labels[m[1]]; ok {
				current = key
				sections[key] = strings.TrimSpace(m[2])
				continue
			}
			current = ""
			continue
		}
		if current != "" {
			sections[current] += "\n" + line
		}
	}
	// Initial format profile: header, database identity, check block, and script footer.
	// Missing selected sections also invalidate the assessment rather than implying success.
	// Linux scripts may omit the Script Info heading but provide Script and Run by.
	// A heading alone does not prove the report reached the footer. Both Windows
	// and Linux layouts must include a usable Run by value; Linux also proves the
	// footer with its Script metadata because it may omit the Script Info heading.
	hasFooter := p.Hostname != "" && (scriptHeading || scriptFooter)
	p.Complete = p.ReportTime != "" && databaseNameRE.MatchString(body) && strings.Contains(body, "Database checks") && hasFooter && len(sections) > 0
	for key, selected := range settings.Selected {
		if selected && key != "missing_email" {
			if _, ok := sections[key]; !ok {
				p.Complete = false
			}
		}
	}
	if !p.Complete {
		status := "not_evaluated"
		if settings.Selected["missing_email"] {
			status = "failed"
		}
		p.Results = append(p.Results, models.CheckResult{CheckType: "missing_email", Status: status, Summary: "Received but incomplete", Evidence: body})
		return p
	}
	for _, key := range []string{"datafiles", "backups", "tablespace", "segments", "extents", "failed_jobs", "indexes", "invalid_objects", "archive_destinations", "fra", "asm_space", "clusterware", "max_lag", "filesystem"} {
		text, exists := sections[key]
		if !exists {
			continue
		}
		text = strings.TrimSpace(text)
		r := models.CheckResult{CheckType: key, Evidence: text, Status: "not_evaluated", Summary: "Check not selected"}
		if !settings.Selected[key] {
			p.Results = append(p.Results, r)
			continue
		}
		if key == "backups" && text == "NOT_US" {
			r.Status = "not_managed"
			r.Summary = "Not managed by us"
			p.Results = append(p.Results, r)
			continue
		}
		// TODO(fra): RecoveryAreaSpace=OK! passes when selected. Non-OK output
		// is likely an issue, but await a real failing email before defining its
		// parsing, thresholds, and ticket evidence. Until then, keep non-OK FRA
		// output unknown via the fallback below; do not invent a failure rule.
		if text == "OK!" {
			r.Status = "passed"
			r.Summary = "Report indicates OK"
			p.Results = append(p.Results, r)
			continue
		}
		if key == "tablespace" || key == "asm_space" || key == "filesystem" {
			matched := false
			for _, line := range strings.Split(text, "\n") {
				re := freeRE
				if key == "filesystem" {
					re = filesystemRE
					if linuxFilesystem {
						re = linuxFilesystemRE
					}
				}
				m := re.FindStringSubmatch(strings.Join(strings.Fields(line), " "))
				if len(m) != 3 {
					continue
				}
				matched = true
				value, _ := strconv.ParseFloat(m[2], 64)
				item := models.CheckResult{CheckType: key, Resource: m[1], Evidence: line, Value: &value, Status: "not_evaluated", Summary: "Resource not configured"}
				rule, ok := settings.Resources[key][m[1]]
				if ok {
					if rule.Ignore {
						item.Status = "ignored"
						item.Summary = "Resource ignored"
					} else {
						threshold := rule.MinFreePercent
						failed := false
						if key == "filesystem" {
							threshold = rule.MaxUsedPercent
							if threshold != nil {
								failed = value > *threshold
							}
						} else if threshold != nil {
							failed = value < *threshold
						}
						if threshold != nil {
							item.Threshold = threshold
							item.Status = "passed"
							item.Summary = "Within configured threshold"
							if failed {
								item.Status = "failed"
								item.Summary = fmt.Sprintf("%s breached threshold (%g%%)", m[1], value)
							}
						}
					}
				}
				p.Results = append(p.Results, item)
			}
			if !matched {
				r.Status = "unknown"
				r.Summary = "Unrecognised resource table"
				p.Results = append(p.Results, r)
			}
			continue
		}
		if key == "backups" {
			if strings.Contains(text, "Datafiles needing backup:") {
				evidence := []string{}
				for _, line := range strings.Split(text, "\n") {
					fields := strings.Fields(line)
					if len(fields) < 3 || fields[0] != "RMAN" {
						continue
					}
					rule, configured := settings.Resources[key][fields[1]]
					if configured && !rule.Ignore {
						evidence = append(evidence, line)
					}
				}
				if len(evidence) > 0 {
					r.Status = "failed"
					r.Summary = "Datafiles needing backup"
					r.Evidence = strings.Join(evidence, "\n")
				} else {
					r.Status = "not_evaluated"
					r.Summary = "No configured, non-ignored backup rows"
				}
			} else {
				r.Status = "unknown"
				r.Summary = "Unrecognised backup status"
			}
		} else if key == "archive_destinations" && strings.Contains(text, "Invalid Archive Destinations:") {
			r.Status = "failed"
			r.Summary = "Invalid archive destinations"
		} else {
			r.Status = "unknown"
			r.Summary = "Rule for this report output is not implemented"
		}
		p.Results = append(p.Results, r)
	}
	return p
}
