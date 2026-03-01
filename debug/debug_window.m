#import <Cocoa/Cocoa.h>
#include "debug_window.h"

extern void GoSendNotify(char *topic, char *payload);

// ── Layout ────────────────────────────────────────────────────────────────────

static const CGFloat kWinW    = 1120.0;
static const CGFloat kWinH    =  720.0;
static const CGFloat kSideW   =  276.0;  // right sidebar (fixed width)
static const CGFloat kTopBarH =   46.0;  // search bar just below the title bar
static const CGFloat kStatusH =   26.0;  // status strip at the very bottom
#define kLogW (kWinW - kSideW)

// ── Color palette (all dark) ──────────────────────────────────────────────────
// Prefixed `dc` to avoid clashing with AERegistry.h OSType constants.

static NSColor *dcLogBG()   { return [NSColor colorWithWhite:0.06 alpha:1]; }
static NSColor *dcSideBG()  { return [NSColor colorWithWhite:0.11 alpha:1]; }
static NSColor *dcBarBG()   { return [NSColor colorWithWhite:0.09 alpha:1]; }
static NSColor *dcDivider() { return [NSColor colorWithWhite:0.20 alpha:1]; }
static NSColor *dcSecHdr()  { return [NSColor colorWithWhite:0.55 alpha:1]; }
static NSColor *dcMuted()   { return [NSColor colorWithWhite:0.62 alpha:1]; }
static NSColor *dcFg()      { return [NSColor colorWithWhite:0.82 alpha:1]; }
static NSColor *dcDim()     { return [NSColor colorWithWhite:0.46 alpha:1]; }

// Log-line accent colors
static NSColor *dcError() { return [NSColor colorWithRed:1.00 green:0.27 blue:0.23 alpha:1]; }
static NSColor *dcWarn()  { return [NSColor colorWithRed:1.00 green:0.82 blue:0.05 alpha:1]; }
static NSColor *dcInfo()  { return [NSColor colorWithRed:0.38 green:0.82 blue:1.00 alpha:1]; }

// ── Line classification ───────────────────────────────────────────────────────

typedef NS_ENUM(NSInteger, LineType) { LTNormal, LTInfo, LTWarn, LTError };

static LineType classifyLine(NSString *s) {
    if ([s containsString:@"error"] || [s containsString:@"Error"] ||
        [s containsString:@"fatal"] || [s containsString:@"Fatal"])
        return LTError;
    if ([s containsString:@"warn"]  || [s containsString:@"Warn"]  ||
        [s containsString:@"drop"]  || [s containsString:@"WARN"])
        return LTWarn;
    if ([s containsString:@"gpio:"] || [s containsString:@"db:"]     ||
        [s containsString:@"ble:"]  || [s containsString:@"script:"] ||
        [s containsString:@"service="] || [s containsString:@"booting"])
        return LTInfo;
    return LTNormal;
}

static NSColor *colorForType(LineType t) {
    switch (t) {
        case LTError: return dcError();
        case LTWarn:  return dcWarn();
        case LTInfo:  return dcInfo();
        default:      return dcFg();
    }
}

// ── Globals ───────────────────────────────────────────────────────────────────

static NSTextView    *gLogView       = nil;
static NSTextField   *gTopicField    = nil;
static NSTextView    *gPayloadView   = nil;
static NSSearchField *gSearchField   = nil;
static NSTextField   *gStatusLabel   = nil;
static NSTextField   *gStatLines     = nil;
static NSTextField   *gStatErrors    = nil;
static NSTextField   *gStatWarns     = nil;
static NSButton      *gAutoScrollBtn = nil;

static NSMutableArray<NSString *> *gAllLines = nil;
static NSMutableArray<NSNumber *> *gAllTypes = nil;
static NSUInteger gErrCount  = 0;
static NSUInteger gWarnCount = 0;
static BOOL       gAutoScroll = YES;

static NSDateFormatter *gDateFmt = nil;

@class DebugDelegate;
static DebugDelegate *gDelegate = nil;

// ── Helpers ───────────────────────────────────────────────────────────────────

static NSFont *dcMono(CGFloat size) {
    return [NSFont fontWithName:@"Menlo" size:size]
        ?: [NSFont monospacedSystemFontOfSize:size weight:NSFontWeightRegular];
}

static NSString *nowString(void) {
    if (!gDateFmt) {
        gDateFmt = [[NSDateFormatter alloc] init];
        [gDateFmt setDateFormat:@"HH:mm:ss.SSS"];
    }
    return [gDateFmt stringFromDate:[NSDate date]];
}

// Attributed string: dim timestamp prefix + colored message body.
static NSAttributedString *styledLine(NSString *line, LineType type) {
    NSMutableAttributedString *as = [[NSMutableAttributedString alloc] init];
    NSDictionary *base = @{NSFontAttributeName: dcMono(12.5)};

    NSRange br = [line rangeOfString:@"] "];
    NSUInteger split = (br.location != NSNotFound) ? br.location + br.length : 0;

    if (split > 0) {
        NSMutableDictionary *ta = [base mutableCopy];
        ta[NSForegroundColorAttributeName] = dcDim();
        [as appendAttributedString:[[NSAttributedString alloc]
            initWithString:[line substringToIndex:split] attributes:ta]];
    }

    NSMutableDictionary *ma = [base mutableCopy];
    ma[NSForegroundColorAttributeName] = colorForType(type);
    NSString *msg = split > 0 ? [line substringFromIndex:split] : line;
    [as appendAttributedString:[[NSAttributedString alloc]
        initWithString:[msg stringByAppendingString:@"\n"] attributes:ma]];

    return as;
}

// ── Solid-color view ──────────────────────────────────────────────────────────

@interface FillView : NSView
@property (strong) NSColor *fillColor;
@end
@implementation FillView
- (void)drawRect:(NSRect)r { [_fillColor setFill]; NSRectFill(r); }
@end

// ── DebugDelegate ─────────────────────────────────────────────────────────────

@interface DebugDelegate : NSObject <NSApplicationDelegate>
- (void)updateStats;
- (void)updateStatus;
@end

@implementation DebugDelegate

- (void)applicationDidFinishLaunching:(NSNotification *)_ {
    gAllLines = [[NSMutableArray alloc] init];
    gAllTypes = [[NSMutableArray alloc] init];

    NSWindow *win = [[NSWindow alloc]
        initWithContentRect:NSMakeRect(0, 0, kWinW, kWinH)
        styleMask:(NSWindowStyleMaskTitled      |
                   NSWindowStyleMaskClosable    |
                   NSWindowStyleMaskResizable   |
                   NSWindowStyleMaskMiniaturizable)
        backing:NSBackingStoreBuffered
        defer:NO];
    [win setTitle:@"dBike \u2014 Debug Console"];
    [win setMinSize:NSMakeSize(840, 500)];
    [win setAppearance:[NSAppearance appearanceNamed:NSAppearanceNameDarkAqua]];
    [win center];

    NSView *cv = [win contentView];

    // ── Log scroll view (left pane) ───────────────────────────────────────────
    CGFloat logH = kWinH - kTopBarH - kStatusH;
    NSScrollView *logScroll = [[NSScrollView alloc]
        initWithFrame:NSMakeRect(0, kStatusH, kLogW, logH)];
    [logScroll setAutoresizingMask:NSViewWidthSizable | NSViewHeightSizable];
    [logScroll setHasVerticalScroller:YES];
    [logScroll setHasHorizontalScroller:NO];
    [logScroll setBackgroundColor:dcLogBG()];

    // Standard terminal-style NSTextView setup inside a scroll view.
    gLogView = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, kLogW, logH)];
    [gLogView setEditable:NO];
    [gLogView setSelectable:YES];
    [gLogView setBackgroundColor:dcLogBG()];
    [gLogView setTextContainerInset:NSMakeSize(10, 8)];
    [gLogView setVerticallyResizable:YES];
    [gLogView setHorizontallyResizable:NO];
    [gLogView setMaxSize:NSMakeSize(CGFLOAT_MAX, CGFLOAT_MAX)];
    [[gLogView textContainer] setWidthTracksTextView:YES];
    [[gLogView textContainer] setContainerSize:NSMakeSize(kLogW, CGFLOAT_MAX)];
    [gLogView setAutoresizingMask:NSViewWidthSizable];
    [logScroll setDocumentView:gLogView];
    [cv addSubview:logScroll];

    [cv addSubview:[self buildTopBar]];
    [cv addSubview:[self buildStatusBar]];

    // 1-px vertical divider between log and sidebar
    FillView *div = [[FillView alloc]
        initWithFrame:NSMakeRect(kLogW, kStatusH, 1, kWinH - kStatusH)];
    div.fillColor = dcDivider();
    [div setAutoresizingMask:NSViewMinXMargin | NSViewHeightSizable];
    [cv addSubview:div];

    [cv addSubview:[self buildSidebar]];

    [win makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    [self updateStatus];
}

// ── Top bar ───────────────────────────────────────────────────────────────────

- (NSView *)buildTopBar {
    FillView *bar = [[FillView alloc]
        initWithFrame:NSMakeRect(0, kWinH - kTopBarH, kLogW, kTopBarH)];
    bar.fillColor = dcBarBG();
    [bar setAutoresizingMask:NSViewWidthSizable | NSViewMinYMargin];

    FillView *border = [[FillView alloc] initWithFrame:NSMakeRect(0, 0, kLogW, 1)];
    border.fillColor = dcDivider();
    [border setAutoresizingMask:NSViewWidthSizable];
    [bar addSubview:border];

    CGFloat fh = 22, fy = (kTopBarH - fh) / 2.0;

    gSearchField = [[NSSearchField alloc] initWithFrame:NSMakeRect(12, fy, 280, fh)];
    [gSearchField setAutoresizingMask:NSViewWidthSizable];
    [[gSearchField cell] setPlaceholderString:@"Filter logs\u2026"];
    [gSearchField setTarget:self];
    [gSearchField setAction:@selector(doFilter:)];
    [bar addSubview:gSearchField];

    NSButton *clr = [[NSButton alloc]
        initWithFrame:NSMakeRect(kLogW - 72 - 12, fy - 2, 72, 26)];
    [clr setTitle:@"Clear"];
    [clr setBezelStyle:NSBezelStyleRounded];
    [clr setTarget:self];
    [clr setAction:@selector(doClear:)];
    [clr setAutoresizingMask:NSViewMinXMargin];
    [bar addSubview:clr];

    return bar;
}

// ── Status bar ────────────────────────────────────────────────────────────────

- (NSView *)buildStatusBar {
    FillView *bar = [[FillView alloc] initWithFrame:NSMakeRect(0, 0, kWinW, kStatusH)];
    bar.fillColor = dcBarBG();
    [bar setAutoresizingMask:NSViewWidthSizable];

    FillView *border = [[FillView alloc]
        initWithFrame:NSMakeRect(0, kStatusH - 1, kWinW, 1)];
    border.fillColor = dcDivider();
    [border setAutoresizingMask:NSViewWidthSizable];
    [bar addSubview:border];

    CGFloat lh = 14, ly = (kStatusH - lh) / 2.0;
    gStatusLabel = [self plainLabel:@"" frame:NSMakeRect(12, ly, 500, lh) size:11];
    [gStatusLabel setAutoresizingMask:NSViewWidthSizable];
    [bar addSubview:gStatusLabel];

    gAutoScrollBtn = [[NSButton alloc]
        initWithFrame:NSMakeRect(kWinW - 126 - 12, 2, 126, kStatusH - 4)];
    [gAutoScrollBtn setButtonType:NSButtonTypeToggle];
    [gAutoScrollBtn setBezelStyle:NSBezelStyleRounded];
    [gAutoScrollBtn setTitle:@"Auto-scroll"];
    [gAutoScrollBtn setState:NSControlStateValueOn];
    [gAutoScrollBtn setTarget:self];
    [gAutoScrollBtn setAction:@selector(doToggleAutoScroll:)];
    [gAutoScrollBtn setAutoresizingMask:NSViewMinXMargin];
    [bar addSubview:gAutoScrollBtn];

    return bar;
}

// ── Sidebar ───────────────────────────────────────────────────────────────────

- (NSView *)buildSidebar {
    FillView *sb = [[FillView alloc]
        initWithFrame:NSMakeRect(kLogW + 1, kStatusH, kSideW - 1, kWinH - kStatusH)];
    sb.fillColor = dcSideBG();
    [sb setAutoresizingMask:NSViewMinXMargin | NSViewHeightSizable];

    const CGFloat cx = 14;
    const CGFloat cw = kSideW - 1 - cx * 2;
    CGFloat y = kWinH - kStatusH;  // descend from top of sidebar

    // ── NOTIFY ───────────────────────────────────────────────────────────────
    y -= 20;
    [sb addSubview:[self sectionHeader:@"NOTIFY"
                                 frame:NSMakeRect(cx, y - 13, cw, 13)]];
    y -= 13;

    y -= 10;
    [sb addSubview:[self sideLabel:@"Topic" frame:NSMakeRect(cx, y - 20, 48, 16)]];
    gTopicField = [self sideField:@"e.g. ping"
                            frame:NSMakeRect(cx + 52, y - 22, cw - 52, 22)];
    [sb addSubview:gTopicField];
    y -= 22;

    y -= 10;
    [sb addSubview:[self sideLabel:@"Payload" frame:NSMakeRect(cx, y - 16, 54, 14)]];
    y -= 16;

    y -= 6;
    const CGFloat pvH = 78;
    NSScrollView *psv = [[NSScrollView alloc]
        initWithFrame:NSMakeRect(cx, y - pvH, cw, pvH)];
    [psv setBorderType:NSBezelBorder];
    [psv setHasVerticalScroller:YES];
    [psv setHasHorizontalScroller:NO];
    gPayloadView = [[NSTextView alloc] initWithFrame:[[psv contentView] bounds]];
    [gPayloadView setFont:dcMono(11)];
    [gPayloadView setTextColor:dcFg()];
    [gPayloadView setBackgroundColor:[NSColor colorWithWhite:0.08 alpha:1]];
    [gPayloadView setAutomaticSpellingCorrectionEnabled:NO];
    [gPayloadView setAutomaticQuoteSubstitutionEnabled:NO];
    [gPayloadView setAutomaticDashSubstitutionEnabled:NO];
    [gPayloadView setRichText:NO];
    [gPayloadView insertText:@"{}" replacementRange:NSMakeRange(0, 0)];
    [psv setDocumentView:gPayloadView];
    [sb addSubview:psv];
    y -= pvH;

    y -= 10;
    NSButton *sendBtn = [[NSButton alloc]
        initWithFrame:NSMakeRect(cx, y - 28, cw, 28)];
    [sendBtn setTitle:@"Send Notification"];
    [sendBtn setBezelStyle:NSBezelStyleRounded];
    [sendBtn setTarget:self];
    [sendBtn setAction:@selector(doSend:)];
    [sendBtn setKeyEquivalent:@"\r"];
    [sb addSubview:sendBtn];
    y -= 28;

    y -= 18;
    [sb addSubview:[self hRule:NSMakeRect(cx, y, cw, 1)]];

    // ── PRESETS ───────────────────────────────────────────────────────────────
    y -= 14;
    [sb addSubview:[self sectionHeader:@"PRESETS"
                                 frame:NSMakeRect(cx, y - 13, cw, 13)]];
    y -= 13;

    struct { NSString *label; SEL sel; } presets[] = {
        { @"Ping",       @selector(doPresetPing:)      },
        { @"sim.ready",  @selector(doPresetSimReady:)  },
        { @"ipc.error",  @selector(doPresetIpcError:)  },
    };
    for (int i = 0; i < 3; i++) {
        y -= 8;
        [sb addSubview:[self presetBtn:presets[i].label
                                 frame:NSMakeRect(cx, y - 26, cw, 26)
                                action:presets[i].sel]];
        y -= 26;
    }

    y -= 18;
    [sb addSubview:[self hRule:NSMakeRect(cx, y, cw, 1)]];

    // ── STATS ─────────────────────────────────────────────────────────────────
    y -= 14;
    [sb addSubview:[self sectionHeader:@"STATS"
                                 frame:NSMakeRect(cx, y - 13, cw, 13)]];
    y -= 13;

    struct { NSString *key; NSTextField **out; } statRows[] = {
        { @"Lines",    &gStatLines   },
        { @"Errors",   &gStatErrors  },
        { @"Warnings", &gStatWarns   },
    };
    for (int i = 0; i < 3; i++) {
        y -= 8;
        const CGFloat rh = 17;
        [sb addSubview:[self sideLabel:statRows[i].key
                                 frame:NSMakeRect(cx, y - rh, 66, rh)]];
        *statRows[i].out = [self statValue:@"0"
                                     frame:NSMakeRect(cx + 70, y - rh, cw - 70, rh)];
        [sb addSubview:*statRows[i].out];
        y -= rh;
    }

    return sb;
}

// ── Actions ───────────────────────────────────────────────────────────────────

- (IBAction)doSend:(id)_ {
    NSString *topic = [[gTopicField stringValue]
        stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceCharacterSet]];
    if ([topic length] == 0) return;
    NSString *payload = [[gPayloadView string]
        stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
    if ([payload length] == 0) payload = @"{}";
    GoSendNotify((char *)[topic UTF8String], (char *)[payload UTF8String]);
}

- (void)loadPreset:(NSString *)topic payload:(NSString *)payload {
    [gTopicField setStringValue:topic];
    [[gPayloadView textStorage]
        replaceCharactersInRange:NSMakeRange(0, [[gPayloadView textStorage] length])
                     withString:payload];
    GoSendNotify((char *)[topic UTF8String], (char *)[payload UTF8String]);
}

- (IBAction)doPresetPing:(id)_     { [self loadPreset:@"ping"
                                               payload:@"{\"sequence\":0}"]; }
- (IBAction)doPresetSimReady:(id)_ { [self loadPreset:@"sim.ready"
                                               payload:@"{\"source\":\"go\",\"message\":\"IPC ready\"}"]; }
- (IBAction)doPresetIpcError:(id)_ { [self loadPreset:@"ipc.error"
                                               payload:@"{\"source\":\"go\",\"reason\":\"debug\"}"]; }

- (IBAction)doFilter:(id)_         { [self applyFilter]; }

- (IBAction)doToggleAutoScroll:(id)_ {
    gAutoScroll = ([gAutoScrollBtn state] == NSControlStateValueOn);
    if (gAutoScroll) [gLogView scrollToEndOfDocument:nil];
}

- (IBAction)doClear:(id)_ {
    [[gLogView textStorage]
        deleteCharactersInRange:NSMakeRange(0, [[gLogView textStorage] length])];
    [gAllLines removeAllObjects];
    [gAllTypes removeAllObjects];
    gErrCount = gWarnCount = 0;
    [gSearchField setStringValue:@""];
    [self updateStats];
    [self updateStatus];
}

- (void)applyFilter {
    NSString *pattern = [[gSearchField stringValue] lowercaseString];
    BOOL active = [pattern length] > 0;
    [[gLogView textStorage] beginEditing];
    [[gLogView textStorage]
        setAttributedString:[[NSAttributedString alloc] initWithString:@""]];
    for (NSUInteger i = 0; i < [gAllLines count]; i++) {
        NSString *line = gAllLines[i];
        if (!active || [[line lowercaseString] containsString:pattern]) {
            LineType t = (LineType)[gAllTypes[i] integerValue];
            [[gLogView textStorage] appendAttributedString:styledLine(line, t)];
        }
    }
    [[gLogView textStorage] endEditing];
    if (gAutoScroll) [gLogView scrollToEndOfDocument:nil];
    [self updateStatus];
}

- (void)updateStats {
    [gStatLines  setStringValue:[NSString stringWithFormat:@"%lu",
                                 (unsigned long)[gAllLines count]]];
    [gStatErrors setStringValue:[NSString stringWithFormat:@"%lu",
                                 (unsigned long)gErrCount]];
    [gStatWarns  setStringValue:[NSString stringWithFormat:@"%lu",
                                 (unsigned long)gWarnCount]];
}

- (void)updateStatus {
    NSString *filter = [gSearchField stringValue];
    NSUInteger total = [gAllLines count];
    NSString *status;
    if ([filter length] > 0) {
        NSString *src = [gLogView string];
        NSUInteger shown = [[src componentsSeparatedByString:@"\n"] count];
        if ([src length] > 0 && [src hasSuffix:@"\n"]) shown--;
        status = [NSString stringWithFormat:@"%lu lines \u00B7 filter \u201C%@\u201D \u2014 %lu shown",
                  (unsigned long)total, filter, (unsigned long)shown];
    } else if (gErrCount > 0 || gWarnCount > 0) {
        status = [NSString stringWithFormat:
                  @"%lu lines \u00B7 %lu error%@ \u00B7 %lu warning%@",
                  (unsigned long)total,
                  (unsigned long)gErrCount,  gErrCount  == 1 ? @"" : @"s",
                  (unsigned long)gWarnCount, gWarnCount == 1 ? @"" : @"s"];
    } else {
        status = [NSString stringWithFormat:@"%lu lines", (unsigned long)total];
    }
    [gStatusLabel setStringValue:status];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)_ {
    return YES;
}

// ── Builder helpers ───────────────────────────────────────────────────────────

- (NSTextField *)plainLabel:(NSString *)s frame:(NSRect)r size:(CGFloat)sz {
    NSTextField *f = [[NSTextField alloc] initWithFrame:r];
    [f setStringValue:s];
    [f setBezeled:NO]; [f setDrawsBackground:NO];
    [f setEditable:NO]; [f setSelectable:NO];
    [f setTextColor:dcMuted()];
    [f setFont:[NSFont monospacedSystemFontOfSize:sz weight:NSFontWeightRegular]];
    return f;
}

- (NSTextField *)sideLabel:(NSString *)s frame:(NSRect)r {
    NSTextField *f = [[NSTextField alloc] initWithFrame:r];
    [f setStringValue:s];
    [f setBezeled:NO]; [f setDrawsBackground:NO];
    [f setEditable:NO]; [f setSelectable:NO];
    [f setAlignment:NSTextAlignmentRight];
    [f setTextColor:dcMuted()];
    [f setFont:[NSFont systemFontOfSize:11]];
    return f;
}

- (NSTextField *)sectionHeader:(NSString *)s frame:(NSRect)r {
    NSTextField *f = [[NSTextField alloc] initWithFrame:r];
    [f setStringValue:s];
    [f setBezeled:NO]; [f setDrawsBackground:NO];
    [f setEditable:NO]; [f setSelectable:NO];
    [f setTextColor:dcSecHdr()];
    [f setFont:[NSFont systemFontOfSize:10 weight:NSFontWeightSemibold]];
    return f;
}

- (NSTextField *)statValue:(NSString *)s frame:(NSRect)r {
    NSTextField *f = [[NSTextField alloc] initWithFrame:r];
    [f setStringValue:s];
    [f setBezeled:NO]; [f setDrawsBackground:NO];
    [f setEditable:NO]; [f setSelectable:NO];
    [f setTextColor:dcFg()];
    [f setFont:dcMono(12)];
    return f;
}

- (NSTextField *)sideField:(NSString *)placeholder frame:(NSRect)r {
    NSTextField *f = [[NSTextField alloc] initWithFrame:r];
    [[f cell] setPlaceholderString:placeholder];
    [f setFont:dcMono(11)];
    return f;
}

- (FillView *)hRule:(NSRect)r {
    FillView *v = [[FillView alloc] initWithFrame:r];
    v.fillColor = dcDivider();
    return v;
}

- (NSButton *)presetBtn:(NSString *)title frame:(NSRect)r action:(SEL)action {
    NSButton *b = [[NSButton alloc] initWithFrame:r];
    [b setTitle:title];
    [b setBezelStyle:NSBezelStyleRounded];
    [b setAlignment:NSTextAlignmentLeft];
    [b setFont:dcMono(11.5)];
    [b setTarget:self];
    [b setAction:action];
    return b;
}

@end

// ── C interface ───────────────────────────────────────────────────────────────

void DebugWindowRun(void) {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    gDelegate = [[DebugDelegate alloc] init];
    [NSApp setDelegate:gDelegate];
    [NSApp run];
}

void DebugWindowAppendLine(const char *cline) {
    NSString *raw  = [NSString stringWithUTF8String:cline];
    LineType  type = classifyLine(raw);

    dispatch_async(dispatch_get_main_queue(), ^{
        if (!gLogView) return;

        NSString *line = [NSString stringWithFormat:@"[%@] %@", nowString(), raw];
        [gAllLines addObject:line];
        [gAllTypes addObject:@(type)];
        if (type == LTError) gErrCount++;
        else if (type == LTWarn) gWarnCount++;

        NSString *pattern = [[gSearchField stringValue] lowercaseString];
        BOOL show = ([pattern length] == 0 ||
                     [[line lowercaseString] containsString:pattern]);
        if (show) {
            [[gLogView textStorage] appendAttributedString:styledLine(line, type)];
            if (gAutoScroll) [gLogView scrollToEndOfDocument:nil];
        }

        [gDelegate updateStats];
        [gDelegate updateStatus];
    });
}

void DebugWindowStop(void) {
    dispatch_async(dispatch_get_main_queue(), ^{ [NSApp terminate:nil]; });
}
