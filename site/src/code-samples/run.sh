# Check all rules
semcheck

# Check specific files
semcheck src/auth.go

# Run semcheck on staged files
semcheck -pre-commit
