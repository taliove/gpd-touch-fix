name: Bug Report
description: Report a bug or issue
title: "[Bug] "
labels: ["bug"]

body:
  - type: markdown
    attributes:
      value: |
        Thank you for reporting an issue! Please fill out the form below.

  - type: textarea
    attributes:
      label: Description
      description: A clear and concise description of what the bug is.
      placeholder: Describe the bug...
    validations:
      required: true

  - type: textarea
    attributes:
      label: Steps to Reproduce
      description: Steps to reproduce the behavior
      placeholder: |
        1. ...
        2. ...
        3. ...
    validations:
      required: true

  - type: textarea
    attributes:
      label: Expected Behavior
      description: What you expected to happen
    validations:
      required: true

  - type: textarea
    attributes:
      label: Actual Behavior
      description: What actually happened
    validations:
      required: true

  - type: input
    attributes:
      label: Environment
      description: e.g., Windows 11, Go 1.21, etc.
      placeholder: OS, Go version, etc.
    validations:
      required: false

  - type: textarea
    attributes:
      label: Additional Context
      description: Add any other context about the problem here
    validations:
      required: false
