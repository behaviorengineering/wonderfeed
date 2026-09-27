package main

// parseControlConfigFlag extracts --config <path> from control subcommand args.
func parseControlConfigFlag(args []string) (remaining []string, configPath string) {
	var out []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			configPath = args[i+1]
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out, configPath
}
