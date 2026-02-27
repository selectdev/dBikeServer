void DebugWindowRun(void);              // blocking; run on the main OS thread
void DebugWindowAppendLine(const char *line);
void DebugWindowStop(void);             // dispatches [NSApp terminate] on main queue
