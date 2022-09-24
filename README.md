# OpenSCAP Report Publisher

![Build](https://img.shields.io/github/workflow/status/jritter/openscap-report-publisher/Publish%20(main)?label=Build%20%28main%29&style=flat-square)
[![Go Report Card](https://goreportcard.com/badge/github.com/jritter/openscap-report-publisher?style=flat-square)](https://goreportcard.com/report/github.com/jritter/openscap-report-publisher)
[![Releases](https://img.shields.io/github/v/release/jritter/openscap-report-publisher?style=flat-square&label=Release)](https://github.com/jritter/openscap-report-publisher/releases)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](https://opensource.org/licenses/MIT)

## Example of RHEL8 Scan

This example evaluates a RHEL8 system against a CIS L1 Server Benchmark and outputs an ARF formated report into the file arf.xml.

```bash
oscap xccdf eval --profile xccdf_org.ssgproject.content_profile_cis_server_l1 --results-arf resources/arf.xml /usr/share/xml/scap/ssg/content/ssg-rhel8-ds.xml
```

## Generate a HTML report

This command generates a HTML report from and ARF report.

```bash
oscap xccdf generate report --output resources/report.html resources/arf.xml
```

### Example of passed rule

```xml
<arf:asset-report-collection xmlns:arf="http://scap.nist.gov/schema/asset-reporting-format/1.1" xmlns:core="http://scap.nist.gov/schema/reporting-core/1.1" xmlns:ai="http://scap.nist.gov/schema/asset-identification/1.1">
  <arf:reports>
    <arf:report id="xccdf1">
      <arf:content>
        <TestResult xmlns="http://checklists.nist.gov/xccdf/1.2" id="xccdf_org.open-scap_testresult_xccdf_org.ssgproject.content_profile_cis_server_l1" start-time="2022-07-21T20:51:15+01:00" end-time="2022-07-21T20:51:39+01:00" version="0.1.60" test-system="cpe:/a:redhat:openscap:1.3.6">
          <benchmark href="#scap_org.open-scap_comp_ssg-rhel8-xccdf-1.2.xml" id="xccdf_org.ssgproject.content_benchmark_RHEL-8"/>
          <title>OSCAP Scan Result</title>
          <identity authenticated="false" privileged="false">root</identity>
          <profile idref="xccdf_org.ssgproject.content_profile_cis_server_l1"/>
          <target>hostname.example.local</target>
          <target-address>127.0.0.1</target-address>
          <target-address>0:0:0:0:0:0:0:1</target-address>
          <rule-result idref="xccdf_org.ssgproject.content_rule_configure_crypto_policy" role="full" time="2022-07-21T20:51:16+01:00" severity="high" weight="1.000000">
            <result>pass</result>
            <ident system="https://nvd.nist.gov/cce/index.cfm">CCE-80935-0</ident>
            <check system="http://oval.mitre.org/XMLSchema/oval-definitions-5">
              <check-export export-name="oval:ssg-var_system_crypto_policy:var:1" value-id="xccdf_org.ssgproject.content_value_var_system_crypto_policy"/>
              <check-content-ref name="oval:ssg-configure_crypto_policy:def:1" href="#oval0"/>
            </check>
          </rule-result>
        </TestResult>
      </arf:content>
    </arf:report>
  </arf:reports>
</arf:asset-report-collection>
```

### Example of failed rule

```xml
<arf:asset-report-collection xmlns:arf="http://scap.nist.gov/schema/asset-reporting-format/1.1" xmlns:core="http://scap.nist.gov/schema/reporting-core/1.1" xmlns:ai="http://scap.nist.gov/schema/asset-identification/1.1">
  <arf:reports>
    <arf:report id="xccdf1">
      <arf:content>
        <TestResult xmlns="http://checklists.nist.gov/xccdf/1.2" id="xccdf_org.open-scap_testresult_xccdf_org.ssgproject.content_profile_cis_server_l1" start-time="2022-07-21T20:51:15+01:00" end-time="2022-07-21T20:51:39+01:00" version="0.1.60" test-system="cpe:/a:redhat:openscap:1.3.6">
          <benchmark href="#scap_org.open-scap_comp_ssg-rhel8-xccdf-1.2.xml" id="xccdf_org.ssgproject.content_benchmark_RHEL-8"/>
          <title>OSCAP Scan Result</title>
          <identity authenticated="false" privileged="false">root</identity>
          <profile idref="xccdf_org.ssgproject.content_profile_cis_server_l1"/>
          <target>hostname.example.local</target>
          <target-address>127.0.0.1</target-address>
          <target-address>0:0:0:0:0:0:0:1</target-address>
          <rule-result idref="xccdf_org.ssgproject.content_rule_partition_for_tmp" role="full" time="2022-07-21T20:51:16+01:00" severity="low" weight="1.000000">
            <result>fail</result>
            <ident system="https://nvd.nist.gov/cce/index.cfm">CCE-80851-9</ident>
            <check system="http://oval.mitre.org/XMLSchema/oval-definitions-5">
              <check-content-ref name="oval:ssg-partition_for_tmp:def:1" href="#oval0"/>
            </check>
          </rule-result>
        </TestResult>
      </arf:content>
    </arf:report>
  </arf:reports>
</arf:asset-report-collection>
```

## Some interesting Prom queries

### Aggregate passed vs. not passed results

```promql
count_values("openscap_result", openscap_results)
```

### Percentage of passed checks

```promql
count(openscap_results == 1)/count(openscap_results)*100
```

### Percentage of failed checks

```promql
count(openscap_results == 0)/count(openscap_results)*100
```

## How to Build and Run

### Build Container Image

The Container Image can be built using the existing Containerfile:

```bash
podman build -t quay.io/jritter/openscap-report-publisher:latest .
```

### Run the Container Image

Assuming that the ARF RAW Reports are sitting in resources/arf, the container image can be started as follows using podman:

```bash
podman run -v ./resources/reports:/opt/go/resources/reports:Z -it -p 2112:2112 quay.io/jritter/openscap-report-publisher:latest
```
