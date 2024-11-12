#!/bin/bash

set -o pipefail
shopt -s nullglob

environments=()
log_context=""
log_env=""
all_envs=0
report=0

#
# Functions
#

function print_help() {
  cat <<EOF

Usage: ./helm_lint.sh [OPTIONS] ENVIRONMENTS

Runs the helm linter on one or more environments

Options:
  -h, --help    Shows this usage information
  -r, --report  Generate a final linter report for Github CI
  -a, --all     Run the linter on all environments
EOF
}

# Output a message to the log file and stdout with optional context
function log_output() {
  local message=$1

  # Set context to the calling function unless overridden
  local context=${log_context:-${FUNCNAME[1]}}
  local context_fmt="($context)"

  local env_fmt
  if [[ -n $log_env ]]; then
    env_fmt="[${log_env}] "
  fi

  echo "${env_fmt}${context_fmt}: ${message}" >&2
}

# Check helm is installed, or install it if not
function check_helm() {
  if command -v helm &> /dev/null; then
    echo "Helm found"
  else
    echo "Helm not found, installing"
    curl -sL https://get.helm.sh/helm-v3.2.4-linux-amd64.tar.gz -o helm-v3.2.4-linux-amd64.tar.gz
    tar zxvf helm-v3.2.4-linux-amd64.tar.gz
    sudo chmod +x linux-amd64/helm
    sudo mv linux-amd64/helm /usr/local/bin/helm
    rm -rf linux-amd64
    rm -rf helm-v3.2.4-linux-amd64.tar.gz
    if ! command -v helm &> /dev/null; then
      echo "Helm still not found, quitting"
      return 1
    fi
  fi
}

# Lint a specific environment and store the results
function lint_env() {
  local env=$1
  log_env="${env}"
  log_output "Linting env: ${env}"

  local file output lint_rc
  for path in valueFiles/"${env}"/*.yaml; do
    file=$(basename -- "${path}")

    log_output "Linting file: ${file}"

    # Run helm lint, and read each line of output into log_output
    log_context="helm"
    helm lint . -f "${path}" 2>&1 | \
    while read -r line; do
      log_output "${line}"
    done
    lint_rc=$?
    log_context=""

    if [[ $lint_rc -ne 0 ]]; then
      log_output "File: ${file} - FAIL!"
    else
      log_output "File: ${file} - PASS"
    fi

    # Store lint exit code in global associative array
    env_results["$file"]=${lint_rc}
  done
}

function summarise_env() {
  local env=$1
  local total_count=0 fail_count=0
  for file in "${!env_results[@]}"; do
    total_count=$((total_count + 1))
    if [[ ${env_results[$file]} -ne 0 ]]; then
      fail_count=$((fail_count + 1))
    fi
  done

  log_output "${total_count} chart(s) linted, ${fail_count} chart(s) failed"
  if [[ $fail_count -gt 0 ]]; then
    log_output "Environment lint: FAIL"
    return 1
  else
    log_output "Environment lint: PASS"
  fi
}

function summarise_envs() {
  local has_failures=$1

  log_output ""
  log_output "Environment linting complete"
  log_output "${#environments[@]} environments processed."
  if [[ $has_failures -eq 1 ]]; then
    log_output "FAIL: Some environments failed linting!"
    log_output "Check the full output for more details."
  else
    log_output "PASS: All environments successfully linted!"
  fi
}

# Run linter on multiple environments
function lint_envs() {
  local has_failures=0

  for env in "${environments[@]}"; do
    # Define (or re-define) a global associative array to store linter exit codes
    unset env_results
    declare -gA env_results

    # Lint each environment, store the results in the above array
    lint_env "${env}"

    # Check the global results array and summarise each env
    if ! summarise_env "${env}"; then
      has_failures=1
    fi

    # Exit early unless we want a report
    if [[ $report -ne 1 ]]; then continue; fi

    # If enabled, create a report for each environment
    report_env "${env}"
  done

  # Generate a summary at the end detailing whether there were any failures
  summarise_envs "${has_failures}"

  # Exit early unless we want a report
  if [[ $report -ne 1 ]]; then return; fi

  # Collate each individual environment report into a single final report
  finalise_report "${has_failures}"
}

# Generate a report for the environment on any linting failures
function report_env() {
  local env=$1
  local env_report="valueFiles/${env}/lint_report.md"

  cat << EOF >> "${env_report}"
### ${env}
\`\`\`
EOF

  local total_count=0 fail_count=0 result
  for file in "${!env_results[@]}"; do
    result=${env_results[$file]}

    if [[ $result -ne 0 ]]; then
      fail_count=$((fail_count + 1))
      echo "${file} - lint failed" >> "${env_report}"
    fi
    total_count=$((total_count + 1))
  done

  cat << EOF >> "${env_report}"
${total_count} chart(s) linted, ${fail_count} chart(s) failed
\`\`\`
EOF
}

# Combine each individual
function finalise_report() {
  local has_failures=$1
  local report="lint_report.md"
  local env_report=""

  echo "# Helm Lint Report" > $report
  for env in "${environments[@]}"; do
    env_report="valueFiles/${env}/lint_report.md"
    cat "$env_report" >> $report
  done

  if [[ -n ${GITHUB_SERVER_URL} ]]; then
    local job_link="${GITHUB_SERVER_URL}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}#step:4:1"
    echo "[Full Linter Output](${job_link})" >> $report
  fi

  if [[ $has_failures -eq 1 ]]; then
    cat << EOF >> $report

FAIL: Some charts failed linting!
Check the full output for more details.
EOF
  else
    cat << EOF >> $report

PASS: All charts successfully linted!
EOF
  fi
}

function verify_envs() {
  if [[ "${#environments[@]}" -eq 0 ]]; then
    log_output "Error: No environments specified!"
    return 1
  fi

  for env in "${environments[@]}"; do
    if [[ ! -d "valueFiles/${env}" ]]; then
      log_output "Error: Could not find environment '${env}'"
      return 1
    fi
  done
}

function find_all_envs() {
  for path in valueFiles/*/; do
    env=$(basename -- "${path}")
    environments+=("${env}")
  done
}

while test $# -gt 0; do
  case "$1" in
    -h|--help)
      print_help
      exit 0
      ;;
    -a|--all)
      all_envs=1
      shift
      ;;
    -r|--report)
      report=1
      shift
      ;;
    *)
      break
      ;;
  esac
done

# Cleanup old reports
find . -type f -name "lint_report.md" -delete

# Select environments to lint
if [[ $all_envs -ne 0 ]]; then
  find_all_envs
else
  environments=("$@")
fi

# Check environments exist
if ! verify_envs "${environments[@]}" ; then
  print_help
  exit 1
fi

check_helm
lint_envs "${environments[@]}"
