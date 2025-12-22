name: Feature Request
description: Suggest an idea for this project
title: "[Feature] "
labels: ["enhancement"]

body:
  - type: markdown
    attributes:
      value: |
        Thank you for suggesting a feature! Please fill out the form below.

  - type: textarea
    attributes:
      label: Description
      description: A clear and concise description of the feature you want
      placeholder: Describe the feature...
    validations:
      required: true

  - type: textarea
    attributes:
      label: Motivation
      description: Why do you need this feature? What problem does it solve?
      placeholder: Explain the motivation...
    validations:
      required: true

  - type: textarea
    attributes:
      label: Proposed Solution
      description: Describe how you would like this feature to work
    validations:
      required: false

  - type: textarea
    attributes:
      label: Alternatives Considered
      description: Describe alternative solutions or features you've considered
    validations:
      required: false

  - type: textarea
    attributes:
      label: Additional Context
      description: Add any other context or screenshots
    validations:
      required: false
