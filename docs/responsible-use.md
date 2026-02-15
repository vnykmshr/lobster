---
title: Responsible Use
nav_order: 9
---

# Responsible Use Guidelines

## Introduction

Lobster is a powerful web application load testing tool designed to help developers and QA teams validate application performance. This document outlines the legal, ethical, and technical guidelines for using Lobster responsibly.

**Summary**: Only test systems you own or have explicit written permission to test. Respect rate limits, robots.txt, and privacy. If in doubt, do not proceed.

## Legal Requirements

### Authorization is MANDATORY

You MUST have explicit authorization before testing any system:

✅ **Authorized Use:**
- Your own applications and infrastructure
- Systems where you have written permission from the owner
- Penetration testing engagements with signed contracts
- Bug bounty programs that explicitly allow load testing
- Development and staging environments you control

❌ **Unauthorized Use (ILLEGAL):**
- Testing third-party websites without permission
- Load testing production systems without authorization
- "Testing" competitors' websites or services
- Academic research without proper ethical approval
- "Security testing" without a formal engagement

### Legal Consequences

Unauthorized load testing may violate:

- **Computer Fraud and Abuse Act (CFAA)** in the United States
- **Computer Misuse Act** in the United Kingdom
- **Criminal Code provisions** in other jurisdictions
- **Terms of Service** agreements
- **Cloud provider acceptable use policies**

Penalties can include:
- Criminal prosecution and fines
- Civil lawsuits for damages
- Loss of employment or contracts
- Permanent record affecting future opportunities

**When in doubt, get it in writing.** A simple email confirmation from a system owner can protect you legally.

## Ethical Guidelines

### Core Principles

1. **Minimize Harm**: Configure tests to avoid disrupting legitimate users
2. **Transparency**: Be clear about who you are and what you're testing
3. **Privacy**: Protect any data collected during testing
4. **Respect**: Honor robots.txt, rate limits, and server capacity
5. **Accountability**: Take responsibility for your testing activities

### Ethical Checklist

Before running a load test, ask yourself:

- [ ] Do I have explicit permission to test this system?
- [ ] Have I informed relevant stakeholders (operations, security teams)?
- [ ] Am I testing during appropriate hours to minimize user impact?
- [ ] Have I configured reasonable concurrency and rate limits?
- [ ] Am I testing non-production environments when possible?
- [ ] Do I have a plan to stop testing if issues arise?
- [ ] Am I prepared to share findings responsibly?

## Technical Best Practices

### 1. Respect robots.txt

Lobster respects robots.txt by default:

```bash
# Good: Respect robots.txt directives
lobster -url https://example.com

# Only override with explicit permission
lobster -url https://myapp.com -ignore-robots  # ⚠️ Use only on your own systems
```

**Why it matters**: robots.txt represents the website owner's preferences. Ignoring it without permission is disrespectful and potentially illegal.

### 2. Configure Appropriate Rate Limits

```bash
# Good: Reasonable rate for most applications
lobster -url https://example.com -rate 5 -concurrency 10

# Bad: Aggressive settings that could cause service disruption
lobster -url https://example.com -rate 1000 -concurrency 500  # ⚠️ DoS-like behavior
```

**Guidelines**:
- Start with low concurrency (5-10) and gradually increase
- Monitor server health during testing
- Respect HTTP 429 (Too Many Requests) responses (enabled by default)
- Use dry-run mode first to understand request volume: `-dry-run`

### 3. Test Non-Production Environments

Whenever possible:
- Use staging or QA environments
- Create isolated test environments
- Avoid production testing during peak hours
- Get approval from operations teams before production testing

### 4. Use TLS Certificate Validation

```bash
# Good: Use proper TLS validation
lobster -url https://example.com

# Only disable for internal testing with self-signed certificates
lobster -url https://internal-test.local -insecure-skip-verify  # ⚠️ Development only
```

**Never** disable TLS verification when testing over untrusted networks.

### 5. Set Reasonable Test Durations

```bash
# Good: Short initial tests to validate configuration
lobster -url https://example.com -duration 30s

# Then increase duration as needed
lobster -url https://example.com -duration 10m

# Avoid: Extremely long tests without monitoring
lobster -url https://example.com -duration 24h  # ⚠️ Requires explicit approval
```

### 6. Monitor and Respond

During testing:
- Watch server metrics (CPU, memory, response times)
- Monitor error rates and adjust accordingly
- Be ready to stop testing immediately if issues arise
- Have communication channels open with operations teams

## What NOT to Do

### ❌ Forbidden Activities

1. **Testing Without Authorization**
   - Never test systems you don't own without written permission
   - Don't assume permission based on public accessibility

2. **Denial of Service Attacks**
   - Don't intentionally overwhelm servers to cause outages
   - Don't ignore signs of server distress (errors, timeouts)

3. **Bypassing Security Controls**
   - Don't disable robots.txt compliance without permission
   - Don't ignore rate limiting or IP blocking
   - Don't rotate IPs or use proxies to evade detection

4. **Testing in Production Without Approval**
   - Don't "surprise" your operations team with load tests
   - Don't test during critical business hours without coordination

5. **Exposing Sensitive Data**
   - Don't share test reports containing sensitive URLs or data
   - Don't commit reports with secrets to version control
   - Don't test with real user credentials or production data

6. **Ignoring Server Responses**
   - Don't continue testing when receiving 429 (rate limit) responses
   - Don't ignore 503 (service unavailable) responses
   - Don't disable `-respect-429` without good reason

## Data Privacy and Security

### Handling Test Results

Test reports may contain sensitive information:

**Sensitive Data in Reports:**
- URLs with API keys, tokens, or session IDs in query parameters
- Error messages revealing system internals
- Authentication headers and cookies
- Internal server paths and configurations

**Security Practices:**
- **Generate reports to secure locations** with proper file permissions (Lobster uses 0o600)
- **Sanitize URLs** before sharing reports externally
- **Don't commit reports** to version control systems
- **Delete reports** when no longer needed
- **Encrypt reports** if storing long-term
- **Review before sharing** - redact sensitive information

### Authentication in Tests

When testing authenticated endpoints:

```bash
# Use test accounts, never production user credentials
lobster -url https://example.com -auth-type bearer -auth-token "test_token_12345"

# Good: Create dedicated test users
# Bad: Using real user accounts
```

**Best Practices:**
- Use dedicated test accounts with limited privileges
- Never use real user credentials
- Rotate test credentials regularly
- Don't log credentials in reports or console output
- Use environment variables or secure vaults for credentials

### Privacy Considerations

- Test with synthetic data, not real user information
- Avoid testing pages that process personal data when possible
- Follow GDPR, CCPA, and other privacy regulations
- Obtain consent if testing involves any real user data
- Have a data retention policy for test results

## Reporting Issues Responsibly

If you discover security vulnerabilities or performance issues:

### For Your Own Systems
1. Document findings with evidence
2. Prioritize based on severity
3. Work with teams to remediate
4. Retest after fixes are deployed

### For Third-Party Systems
1. **Stop testing immediately** if you find a vulnerability
2. Follow the organization's security disclosure policy
3. Use responsible disclosure practices:
   - Report privately to security contacts
   - Give reasonable time to fix (typically 90 days)
   - Don't publicly disclose until patched
4. Never exploit vulnerabilities you discover
5. Never demand payment or rewards (unless it's a bug bounty program)

## Consequences of Misuse

Misusing Lobster can result in:

**Legal Consequences:**
- Criminal charges and prosecution
- Civil lawsuits for damages caused
- Permanent criminal record
- Financial penalties and restitution

**Professional Consequences:**
- Termination of employment
- Loss of professional certifications
- Reputation damage
- Difficulty finding future employment

**Technical Consequences:**
- IP address blacklisting
- Network access revocation
- Account terminations
- Service bans

**We do not condone misuse of this tool.** Users are solely responsible for their actions.

## Educational Use

### For Students and Researchers

If using Lobster for academic purposes:

✅ **Appropriate Use:**
- Testing your own course projects
- Lab environments specifically designed for testing
- Research with institutional ethics board approval
- Bug bounty programs that welcome researchers

❌ **Inappropriate Use:**
- Testing university systems without permission ("testing" the school website)
- Research projects without ethics approval
- Testing third-party sites for "academic research"
- Demonstrating capabilities on unauthorized systems

**Get approval first.** Most institutions have ethics boards (IRB) that review research involving systems testing.

## Industry-Specific Considerations

### Financial Services
- Extra scrutiny on testing production systems
- May require security team approval
- Avoid testing during market hours
- Special data privacy requirements

### Healthcare
- HIPAA compliance requirements
- Never test with real patient data
- Requires heightened security measures
- May need legal review

### E-commerce
- Avoid testing during peak sales periods
- Don't test checkout flows without approval
- Be mindful of inventory systems
- Consider impact on analytics

### Government Systems
- May require special authorization
- Higher legal scrutiny
- Stricter security requirements
- Coordinate with security operations centers

## Getting Help

If you're unsure whether your use case is appropriate:

1. **Consult your legal team** - When in doubt, ask your organization's legal counsel
2. **Contact the system owner** - Get written permission via email
3. **Review terms of service** - Check if load testing is explicitly forbidden
4. **Start small** - Begin with minimal impact and scale up with approval
5. **Document everything** - Keep records of approvals and communications

## Updates to These Guidelines

These guidelines may be updated as best practices evolve. Check the [repository](https://github.com/1mb-dev/lobster) for the latest version.

Last updated: 2025

## Summary

Responsible use of Lobster comes down to three simple rules:

1. **Get Permission**: Always have explicit authorization to test
2. **Minimize Harm**: Configure tests to avoid disrupting services or users
3. **Protect Privacy**: Handle test data and results securely

Testing is a valuable practice that makes applications better. Let's do it responsibly and ethically.

---

**Remember**: The fact that you *can* test something doesn't mean you *should*. When in doubt, ask for permission.

If you have questions about these guidelines, please [open an issue](https://github.com/1mb-dev/lobster/issues) or contact the maintainers.
