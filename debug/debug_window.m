#import <Cocoa/Cocoa.h>
#include "debug_window.h"

extern void GoSendNotify(char *topic, char *payload);

// forward declarations for helpers used before their definitions
static NSFont *monoFont(void);
static NSColor *lineColor(NSString *line);

static NSTextView  *gTextView     = nil;
static NSTextField *gTopicField   = nil;
static NSTextField *gPayloadField = nil;
static NSSearchField *gSearchField = nil;          // filter input
static NSMutableArray<NSString*> *gAllLines = nil; // stores every line for filtering

// formatter for timestamps
static NSDateFormatter *gDateFormatter = nil;

// return current time string
static NSString *timeString(void) {
    if (!gDateFormatter) {
        gDateFormatter = [[NSDateFormatter alloc] init];
        [gDateFormatter setDateFormat:@"HH:mm:ss.SSS"];
    }
    return [gDateFormatter stringFromDate:[NSDate date]];
}

static const CGFloat kBarH = 74.0;
static const CGFloat kPad  =  8.0;

@interface DebugDelegate : NSObject <NSApplicationDelegate>
@end

@implementation DebugDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)_ {
    const CGFloat W = 960, H = 680;

    NSWindow *win = [[NSWindow alloc]
        initWithContentRect:NSMakeRect(0, 0, W, H)
        styleMask:(NSWindowStyleMaskTitled      |
                   NSWindowStyleMaskClosable    |
                   NSWindowStyleMaskResizable   |
                   NSWindowStyleMaskMiniaturizable)
        backing:NSBackingStoreBuffered
        defer:NO];
    [win setTitle:@"dBike \u2014 Debug Console"];
    [win setMinSize:NSMakeSize(640, 400)];
    [win center];

    NSView *cv = [win contentView];

    NSScrollView *sv = [[NSScrollView alloc]
        initWithFrame:NSMakeRect(0, kBarH, W, H - kBarH)];
    [sv setAutoresizingMask:NSViewWidthSizable | NSViewHeightSizable];
    [sv setHasVerticalScroller:YES];
    [sv setHasHorizontalScroller:NO];
    [sv setBackgroundColor:[NSColor colorWithWhite:0.09 alpha:1]];

    gTextView = [[NSTextView alloc] initWithFrame:[[sv contentView] bounds]];
    [gTextView setEditable:NO];
    [gTextView setSelectable:YES];
    [gTextView setBackgroundColor:[NSColor colorWithWhite:0.09 alpha:1]];
    [gTextView setTextContainerInset:NSMakeSize(6, 6)];
    [gTextView setAutoresizingMask:NSViewWidthSizable];
    [sv setDocumentView:gTextView];
    [cv addSubview:sv];

    NSView *bar = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, W, kBarH)];
    [bar setAutoresizingMask:NSViewWidthSizable];
    [cv addSubview:bar];

    // initialize line storage for filtering
    gAllLines = [[NSMutableArray alloc] init];

    NSBox *sep = [[NSBox alloc] initWithFrame:NSMakeRect(0, kBarH - 1, W, 1)];
    [sep setBoxType:NSBoxSeparator];
    [sep setAutoresizingMask:NSViewWidthSizable];
    [bar addSubview:sep];

    const CGFloat sendW = 120, r1y = 42, fh = 22;
    // search field on second row
    const CGFloat searchW = 200;

    [bar addSubview:[self labelText:@"Topic"
                             frame:NSMakeRect(kPad, r1y, 38, fh)
                              mask:NSViewMaxXMargin]];

    gTopicField = [self inputPlaceholder:@"topic"
                                   frame:NSMakeRect(50, r1y, 120, fh)
                                    mask:NSViewMaxXMargin];
    [gTopicField setNextKeyView:nil]; // tab moves to payload (set below)
    [bar addSubview:gTopicField];

    [bar addSubview:[self labelText:@"Payload"
                             frame:NSMakeRect(178, r1y, 52, fh)
                              mask:NSViewMaxXMargin]];

    // Payload field grows with the window; right edge stays sendW+2*kPad from right.
    CGFloat pfX = 234, pfW = W - pfX - sendW - 2 * kPad;
    gPayloadField = [self inputPlaceholder:@"{}"
                                     frame:NSMakeRect(pfX, r1y, pfW, fh)
                                      mask:NSViewWidthSizable];
    [gPayloadField setAction:@selector(doSend:)];
    [gPayloadField setTarget:self];
    [bar addSubview:gPayloadField];

    // search field (second row)
    gSearchField = [[NSSearchField alloc] initWithFrame:NSMakeRect(kPad, 10, searchW, fh)];
    [gSearchField setAutoresizingMask:NSViewMaxXMargin | NSViewWidthSizable];
    [gSearchField setTarget:self];
    [gSearchField setAction:@selector(doFilter:)];
    [bar addSubview:gSearchField];

    [gTopicField setNextKeyView:gPayloadField];

    NSButton *sendBtn = [self buttonTitle:@"Send Notify"
                                   action:@selector(doSend:)
                                    frame:NSMakeRect(W - sendW - kPad, r1y - 2, sendW, 26)
                                     mask:NSViewMinXMargin];
    [bar addSubview:sendBtn];
    [win setDefaultButtonCell:[sendBtn cell]];

    const CGFloat r2y = 10, bh = 26;

    // presets dropdown to conserve space
    NSPopUpButton *presetMenu = [[NSPopUpButton alloc]
        initWithFrame:NSMakeRect(kPad, r2y, 120, bh) pullsDown:NO];
    [presetMenu addItemWithTitle:@"Presets"];
    [presetMenu.menu addItemWithTitle:@"Ping" action:@selector(doPresetPing:) keyEquivalent:@""];
    [presetMenu.menu addItemWithTitle:@"sim.ready" action:@selector(doPresetSimReady:) keyEquivalent:@""];
    [presetMenu.menu addItemWithTitle:@"ipc.error" action:@selector(doPresetIpcError:) keyEquivalent:@""];
    [bar addSubview:presetMenu];

    [bar addSubview:[self buttonTitle:@"Clear Log"
                               action:@selector(doClear:)
                                frame:NSMakeRect(W - 90 - kPad, r2y, 90, bh)
                                 mask:NSViewMinXMargin]];

    [win makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
}

// ── Actions ───────────────────────────────────────────────────────────────

- (IBAction)doSend:(id)_ {
    NSString *topic = [[gTopicField stringValue]
        stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceCharacterSet]];
    if ([topic length] == 0) return;

    NSString *payload = [[gPayloadField stringValue]
        stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceCharacterSet]];
    if ([payload length] == 0) payload = @"{}";

    GoSendNotify((char *)[topic UTF8String], (char *)[payload UTF8String]);
}

- (IBAction)doPresetPing:(id)_ {
    [gTopicField setStringValue:@"ping"];
    [gPayloadField setStringValue:@"{\"sequence\":0}"];
    GoSendNotify("ping", "{\"sequence\":0}");
}

- (IBAction)doPresetSimReady:(id)_ {
    [gTopicField setStringValue:@"sim.ready"];
    [gPayloadField setStringValue:@"{\"source\":\"go\",\"message\":\"IPC ready\"}"];
    GoSendNotify("sim.ready", "{\"source\":\"go\",\"message\":\"IPC ready\"}");
}

- (IBAction)doPresetIpcError:(id)_ {
    [gTopicField setStringValue:@"ipc.error"];
    [gPayloadField setStringValue:@"{\"source\":\"go\",\"reason\":\"debug\"}"];
    GoSendNotify("ipc.error", "{\"source\":\"go\",\"reason\":\"debug\"}");
}

- (IBAction)doFilter:(id)_ {
    [self applyFilter];
}

- (void)applyFilter {
    NSString *pattern = [[gSearchField stringValue] lowercaseString];
    [[gTextView textStorage] beginEditing];
    [[gTextView textStorage] setAttributedString:[[NSAttributedString alloc] initWithString:@""]];
    for (NSString *line in gAllLines) {
        if ([pattern length] == 0 || [[line lowercaseString] containsString:pattern]) {
            NSDictionary *attrs = @{NSFontAttributeName: monoFont(),
                                     NSForegroundColorAttributeName: lineColor(line)};
            [[gTextView textStorage] appendAttributedString:
                [[NSAttributedString alloc] initWithString:[line stringByAppendingString:@"\n"]
                                                attributes:attrs]];
        }
    }
    [[gTextView textStorage] endEditing];
    [gTextView scrollToEndOfDocument:nil];
}

- (IBAction)doClear:(id)_ {
    if (gTextView) {
        [[gTextView textStorage]
            deleteCharactersInRange:NSMakeRange(0, [[gTextView textStorage] length])];
    }
    [gAllLines removeAllObjects];
    if (gSearchField) {
        [gSearchField setStringValue:@""];
    }
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)_ {
    return YES;
}

- (NSTextField *)labelText:(NSString *)s
                     frame:(NSRect)r
                      mask:(NSAutoresizingMaskOptions)m {
    NSTextField *f = [[NSTextField alloc] initWithFrame:r];
    [f setStringValue:s];
    [f setBezeled:NO]; [f setDrawsBackground:NO];
    [f setEditable:NO]; [f setSelectable:NO];
    [f setAlignment:NSTextAlignmentRight];
    [f setTextColor:[NSColor secondaryLabelColor]];
    [f setFont:[NSFont systemFontOfSize:11]];
    [f setAutoresizingMask:m];
    return f;
}

- (NSTextField *)inputPlaceholder:(NSString *)ph
                            frame:(NSRect)r
                             mask:(NSAutoresizingMaskOptions)m {
    NSTextField *f = [[NSTextField alloc] initWithFrame:r];
    [[f cell] setPlaceholderString:ph];
    NSFont *mono = [NSFont fontWithName:@"Menlo" size:11]
                ?: [NSFont monospacedSystemFontOfSize:11 weight:NSFontWeightRegular];
    [f setFont:mono];
    [f setAutoresizingMask:m];
    return f;
}

- (NSButton *)buttonTitle:(NSString *)title
                   action:(SEL)action
                    frame:(NSRect)r
                     mask:(NSAutoresizingMaskOptions)m {
    NSButton *b = [[NSButton alloc] initWithFrame:r];
    [b setTitle:title];
    [b setBezelStyle:NSBezelStyleRounded];
    [b setTarget:self];
    [b setAction:action];
    [b setAutoresizingMask:m];
    return b;
}

@end

static NSFont *monoFont(void) {
    NSFont *f = [NSFont fontWithName:@"Menlo" size:12];
    return f ?: [NSFont monospacedSystemFontOfSize:12 weight:NSFontWeightRegular];
}

static NSColor *lineColor(NSString *line) {
    if ([line containsString:@"error"] || [line containsString:@"Error"])
        return [NSColor colorWithRed:1.0 green:0.35 blue:0.30 alpha:1];
    if ([line containsString:@"warn"] || [line containsString:@"drop"])
        return [NSColor colorWithRed:1.0 green:0.78 blue:0.20 alpha:1];
    return [NSColor colorWithRed:0.18 green:0.83 blue:0.28 alpha:1];
}

void DebugWindowRun(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    DebugDelegate *d = [[DebugDelegate alloc] init];
    [NSApp setDelegate:d];
    [NSApp run];
}

void DebugWindowAppendLine(const char *cline) {
    NSString *line = [NSString stringWithUTF8String:cline];
    // prefix timestamp
    NSString *timed = [NSString stringWithFormat:@"[%@] %@", timeString(), line];

    dispatch_async(dispatch_get_main_queue(), ^{
        if (!gTextView) return;

        // keep the full history for filtering
        [gAllLines addObject:timed];

        NSString *pattern = [[gSearchField stringValue] lowercaseString];
        BOOL show = ([pattern length] == 0 || [[timed lowercaseString] containsString:pattern]);
        if (show) {
            NSDictionary *attrs = @{NSFontAttributeName:            monoFont(),
                                    NSForegroundColorAttributeName: lineColor(timed),};
            [[gTextView textStorage] appendAttributedString:
                [[NSAttributedString alloc]
                    initWithString:[timed stringByAppendingString:@"\n"]
                        attributes:attrs]];
            [gTextView scrollToEndOfDocument:nil];
        }
    });
}

void DebugWindowStop(void) {
    dispatch_async(dispatch_get_main_queue(), ^{ [NSApp terminate:nil]; });
}
