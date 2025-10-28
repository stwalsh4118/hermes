# Virtual TV Design System

**Version 1.0** | Last Updated: October 28, 2025

---

## Table of Contents

1. [Design Philosophy](#design-philosophy)
2. [Color System](#color-system)
3. [Typography](#typography)
4. [Spacing & Layout](#spacing--layout)
5. [Border & Shadow System](#border--shadow-system)
6. [Component Patterns](#component-patterns)
7. [Animations & Transitions](#animations--transitions)
8. [Layout Components](#layout-components)
9. [State Patterns](#state-patterns)
10. [Usage Guidelines](#usage-guidelines)
11. [Theme System](#theme-system)

---

## Design Philosophy

### Core Concept

Virtual TV combines **nostalgic 80s/90s Saturday morning cartoon aesthetics** with the sophistication of modern streaming services. The design evokes memories of vintage TV sets, VCR displays, and arcade machines while maintaining usability and accessibility standards.

### Key Principles

1. **Chunky, Tactile Interfaces**
   - Bold borders (2px-4px) create clear visual hierarchy
   - Offset shadows provide depth and physicality
   - Components feel "pressable" and responsive to interaction

2. **VCR/CRT Screen Aesthetics**
   - Scanline effects on media content
   - Subtle screen glow for atmospheric depth
   - Flickering animations for authenticity

3. **Retro Color Palettes**
   - Warm tones inspired by vintage TV sets
   - Electric colors from arcade machines
   - High saturation for visual impact

4. **High-Contrast Typography**
   - Monospace fonts for technical/retro feel
   - Bold weights for readability
   - Uppercase text for emphasis and style

5. **Playful but Functional**
   - Interactions feel satisfying (button press animations)
   - Personality without sacrificing usability
   - Nostalgic without being kitsch

---

## Color System

### Theme Architecture

Virtual TV uses **OKLCH color space** for perceptual uniformity across all themes. This ensures consistent lightness and chroma regardless of hue, making themes feel balanced and professional.

**Available Themes:**
- Modern Clean (Dark) - Default streaming service aesthetic
- Retro - Warm 80s/90s Saturday morning cartoons
- Neon Arcade - Electric purple/magenta with cyan
- Sunset VHS - Coral/salmon pink with purple
- Retro Gaming - Lime green with magenta
- Miami Vice - Hot pink with turquoise

### Core Design Tokens

All themes implement the following semantic tokens:

#### Surface Colors
```css
--background       /* Main application background */
--foreground       /* Primary text color */
--card             /* Card/panel background */
--card-foreground  /* Text on cards */
--popover          /* Dropdown/modal background */
--popover-foreground /* Text on popovers */
```

#### Semantic Colors
```css
--primary              /* Brand color, main CTAs */
--primary-foreground   /* Text on primary color */
--secondary            /* Secondary actions/content */
--secondary-foreground /* Text on secondary color */
--accent               /* Highlights, secondary CTAs */
--accent-foreground    /* Text on accent color */
--destructive          /* Errors, delete actions */
--destructive-foreground /* Text on destructive color */
--muted                /* Disabled states, subtle backgrounds */
--muted-foreground     /* Secondary text, labels */
```

#### UI Element Colors
```css
--border  /* Standard border color */
--input   /* Form input borders/backgrounds */
--ring    /* Focus ring color */
```

#### Special Effects
```css
--glass-bg        /* Glassmorphism background */
--glass-border    /* Glassmorphism border */
--glass-blur      /* Blur amount for glass effect */
--crt-glow        /* CRT screen glow color */
--scanline-color  /* Scanline overlay color */
```

### Theme Values

#### Modern Clean (Dark)
```css
.dark {
  --background: oklch(0.1 0.01 264);
  --foreground: oklch(0.98 0 0);
  --primary: oklch(0.65 0.25 264);
  --accent: oklch(0.7 0.28 330);
  --destructive: oklch(0.6 0.25 27);
  --crt-glow: oklch(0.65 0.25 264 / 0.3);
}
```

#### Retro Theme
```css
.retro {
  --background: oklch(0.92 0.02 60);  /* Warm beige/cream */
  --foreground: oklch(0.2 0.03 30);
  --primary: oklch(0.6 0.15 200);     /* Teal/cyan */
  --secondary: oklch(0.75 0.12 50);   /* Warm orange */
  --accent: oklch(0.65 0.22 340);     /* Hot pink/magenta */
  --destructive: oklch(0.55 0.22 30);
  --crt-glow: oklch(0.6 0.15 200 / 0.2);
}
```

#### Neon Arcade
```css
.neon-arcade {
  --background: oklch(0.15 0.03 280);  /* Deep navy/black */
  --foreground: oklch(0.98 0.02 280);
  --primary: oklch(0.65 0.28 320);     /* Electric purple */
  --secondary: oklch(0.7 0.18 200);    /* Bright cyan */
  --accent: oklch(0.8 0.18 90);        /* Yellow/gold */
  --destructive: oklch(0.6 0.28 350);  /* Hot pink */
}
```

#### Sunset VHS
```css
.sunset-vhs {
  --background: oklch(0.18 0.02 30);   /* Dark charcoal */
  --foreground: oklch(0.95 0.02 40);
  --primary: oklch(0.7 0.18 20);       /* Coral/salmon pink */
  --secondary: oklch(0.5 0.2 300);     /* Deep purple */
  --accent: oklch(0.75 0.15 70);       /* Warm amber/gold */
}
```

#### Retro Gaming
```css
.retro-gaming {
  --background: oklch(0.12 0.04 300);  /* Dark purple/black */
  --foreground: oklch(0.95 0.02 120);
  --primary: oklch(0.75 0.22 130);     /* Lime green */
  --secondary: oklch(0.65 0.28 330);   /* Hot magenta */
  --accent: oklch(0.7 0.2 50);         /* Bright orange */
}
```

#### Miami Vice
```css
.miami-vice {
  --background: oklch(0.14 0.04 260);  /* Dark navy */
  --foreground: oklch(0.96 0.02 320);
  --primary: oklch(0.68 0.26 340);     /* Hot pink */
  --secondary: oklch(0.55 0.24 290);   /* Deep purple */
  --accent: oklch(0.72 0.16 190);      /* Turquoise/aqua */
}
```

---

## Typography

### Font Stack

```css
--font-sans: "Geist", "Geist Fallback";
--font-mono: "Geist Mono", "Geist Mono Fallback";
```

**Geist** is used for the sans-serif family, providing excellent readability and modern geometric forms. **Geist Mono** is used extensively throughout the interface to reinforce the retro/technical aesthetic.

### Text Utility Classes

#### VCR Text (`.vcr-text`)
Signature retro text style for headings and important UI elements.

```css
.vcr-text {
  font-family: var(--font-mono);
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  text-shadow: 2px 2px 0 var(--crt-glow);
}
```

**Usage:**
- Page titles
- Major section headings
- Call-to-action buttons
- Brand elements

**Example:**
```tsx
<h1 className="text-4xl font-bold vcr-text">Your Channels</h1>
```

#### Retro Button (`.retro-button`)
Custom button styling for chunky, tactile buttons. See [Component Patterns](#component-patterns) for full details.

### Type Scale

| Element | Class | Size | Usage |
|---------|-------|------|-------|
| Hero Heading | `text-4xl` | 2.25rem (36px) | Page titles, hero sections |
| Page Heading | `text-3xl` | 1.875rem (30px) | Main page headings |
| Section Heading | `text-2xl` | 1.5rem (24px) | Section titles |
| Large Body | `text-lg` | 1.125rem (18px) | Descriptions, subtitles |
| Body | `text-base` | 1rem (16px) | Standard body text |
| Small | `text-sm` | 0.875rem (14px) | Labels, captions |
| Extra Small | `text-xs` | 0.75rem (12px) | Badges, metadata |

### Font Weights

- `font-normal` (400) - Rarely used, reserved for long-form content
- `font-bold` (700) - **Primary weight**, used for most UI text
- Retro aesthetic favors bold, high-contrast text

### Best Practices

1. **Use monospace fonts** (`font-mono`) for:
   - All body text and UI labels
   - Technical information (file paths, stats)
   - Data tables
   - Form inputs

2. **Use `.vcr-text`** for:
   - Page titles
   - Major calls-to-action
   - Brand elements
   - Emphasis text

3. **Always uppercase** for:
   - Buttons
   - Navigation items
   - Status labels
   - Action text

---

## Spacing & Layout

### Container Pattern

Standard page container with responsive padding:

```tsx
<main className="container mx-auto px-6 py-8">
  {/* Page content */}
</main>
```

- `container` - Responsive max-width with auto margins
- `px-6` - 1.5rem horizontal padding
- `py-8` - 2rem vertical padding

### Grid Patterns

#### Channel/Card Grid
```tsx
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
  {/* Cards */}
</div>
```
- Mobile: Single column
- Tablet: 2 columns
- Desktop: 3 columns
- Gap: 2rem (32px)

#### Stats Grid
```tsx
<div className="grid grid-cols-4 gap-6">
  {/* Stat cards */}
</div>
```
- Fixed 4 columns (consider responsive breakpoints for production)
- Gap: 1.5rem (24px)

#### Media Library Grid
```tsx
<div className="grid grid-cols-3 gap-6">
  {/* Media items */}
</div>
```
- Fixed 3 columns
- Gap: 1.5rem (24px)

### Section Spacing

| Purpose | Class | Size | Usage |
|---------|-------|------|-------|
| Major sections | `mb-12` | 3rem (48px) | Hero to content, major page divisions |
| Standard sections | `mb-8` | 2rem (32px) | Between content sections |
| Section header | `mb-6` | 1.5rem (24px) | Header to content |
| Component groups | `gap-4` | 1rem (16px) | Form fields, button groups |
| Component groups | `gap-6` | 1.5rem (24px) | Card grids, stats |
| Component groups | `gap-8` | 2rem (32px) | Major card grids |

---

## Border & Shadow System

### Border Weights

The border system creates **visual hierarchy through thickness**:

#### Standard Emphasis (`border-2`)
**2px borders** for:
- Buttons
- Form inputs
- Status badges
- Secondary UI elements

```tsx
<button className="border-2 border-primary">Action</button>
```

#### Heavy Emphasis (`border-4`)
**4px borders** for:
- Cards and panels
- Tables
- Major containers
- Page sections

```tsx
<div className="border-4 border-primary/30">
  {/* Card content */}
</div>
```

### Border Color Patterns

#### Subtle Borders
For non-interactive containers:
```tsx
border-primary/20   /* Very subtle, 20% opacity */
border-primary/30   /* Subtle, 30% opacity */
```

#### Semantic Borders
For interactive or semantic elements:
```tsx
border-primary      /* Full opacity for emphasis */
border-destructive  /* Error states */
border-accent       /* Highlighted items */
border-border       /* Default theme border color */
```

### Shadow System (Retro Offset Shadows)

Virtual TV uses **flat, offset shadows** instead of blur shadows to create a retro, 2D aesthetic reminiscent of old print graphics.

#### Standard Card Shadow
```tsx
shadow-[8px_8px_0_rgba(0,0,0,0.2)]
```
- 8px right offset
- 8px down offset
- 0 blur
- 20% black

**Hover state:**
```tsx
hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)]
```
Creates a "pressing in" effect by reducing offset.

#### Heavy Shadow (Containers)
```tsx
shadow-[8px_8px_0_rgba(0,0,0,0.6)]
```
- Same offset, darker (60% black)
- For tables, major containers
- Creates strong depth

#### Button Shadow Pattern
```tsx
shadow-[6px_6px_0_rgba(0,0,0,0.2)] 
hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)]
```
- Smaller offset for buttons (6px → 3px)
- Creates satisfying press effect

#### Pressed State
```tsx
shadow-[2px_2px_0_rgba(0,0,0,0.2)]
```
- Minimal offset for pressed/active state
- Can go to `0` for fully pressed

### Complete Shadow Examples

**Standard card:**
```tsx
<div className="
  rounded-xl 
  border-4 border-primary/20 
  shadow-[8px_8px_0_rgba(0,0,0,0.2)] 
  hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)]
  transition-all
">
  {/* Content */}
</div>
```

**Table container:**
```tsx
<div className="
  rounded-xl 
  border-4 border-primary 
  shadow-[8px_8px_0_rgba(0,0,0,0.6)]
  overflow-hidden
">
  <table>{/* Table content */}</table>
</div>
```

---

## Component Patterns

### Retro Button

The signature button style with chunky borders and press animation.

#### CSS Definition
```css
.retro-button {
  border: 3px solid currentColor;
  font-weight: 700;
  text-transform: uppercase;
  box-shadow: 4px 4px 0 currentColor;
  letter-spacing: 0.05em;
  transition: all 0.1s ease;
}

.retro-button:hover {
  transform: translate(2px, 2px);
  box-shadow: 2px 2px 0 currentColor;
}

.retro-button:active {
  transform: translate(4px, 4px);
  box-shadow: 0 0 0 currentColor;
}
```

#### Usage Patterns

**Primary button:**
```tsx
<button className="
  retro-button 
  bg-primary text-primary-foreground 
  hover:bg-primary/80 
  px-6 py-3 rounded-lg font-bold 
  border-2 border-primary-foreground/20 
  shadow-[6px_6px_0_rgba(0,0,0,0.2)] 
  hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] 
  transition-all
">
  CREATE CHANNEL
</button>
```

**Secondary button:**
```tsx
<button className="
  retro-button 
  bg-muted text-foreground 
  hover:bg-muted/60 
  px-4 py-2 rounded-lg font-bold text-sm 
  border-2 border-primary/30 
  shadow-[6px_6px_0_rgba(0,0,0,0.2)] 
  hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] 
  transition-all
">
  CHANNELS
</button>
```

**Accent button:**
```tsx
<button className="
  retro-button 
  bg-accent text-accent-foreground 
  hover:bg-accent/80 
  px-4 py-2 rounded-lg font-bold text-sm 
  border-2 border-accent-foreground/20 
  shadow-[6px_6px_0_rgba(0,0,0,0.2)] 
  hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] 
  transition-all
">
  EDIT
</button>
```

**Destructive button:**
```tsx
<button className="
  retro-button 
  bg-destructive/20 text-destructive 
  hover:bg-destructive/40 
  px-4 py-2 rounded-lg font-bold text-sm 
  border-2 border-destructive 
  shadow-[6px_6px_0_rgba(0,0,0,0.2)] 
  hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] 
  transition-all
">
  DELETE
</button>
```

### CRT Screen Effect

Applies vintage CRT monitor aesthetic to media content.

#### CSS Definition
```css
.crt-screen {
  position: relative;
  border-radius: 0.5rem;
  overflow: hidden;
  box-shadow: 
    inset 0 0 40px var(--crt-glow), 
    0 0 60px var(--crt-glow), 
    0 8px 24px rgba(0, 0, 0, 0.3);
}

/* Horizontal scanlines */
.crt-screen::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    to bottom, 
    transparent 50%, 
    var(--scanline-color) 50%
  );
  background-size: 100% 4px;
  pointer-events: none;
  z-index: 2;
  animation: crt-flicker 0.15s infinite;
}

/* Moving scanline */
.crt-screen::after {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 100%;
  background: linear-gradient(
    to bottom, 
    transparent, 
    var(--scanline-color) 50%, 
    transparent
  );
  pointer-events: none;
  z-index: 3;
  animation: scanline 8s linear infinite;
  opacity: 0.5;
}
```

#### Usage
Apply to video thumbnails and media preview areas:

```tsx
<div className="crt-screen aspect-video bg-muted">
  <img src={thumbnail} alt={title} className="w-full h-full object-cover" />
</div>
```

**When to use:**
- Video/media thumbnails
- Channel preview areas
- Content display areas
- **Not** on UI chrome (buttons, headers, etc.)

### Glass Effect

Subtle glassmorphism for overlays and floating elements.

#### CSS Definition
```css
.glass {
  background: var(--glass-bg);
  backdrop-filter: blur(var(--glass-blur));
  -webkit-backdrop-filter: blur(var(--glass-blur));
  border: 1px solid var(--glass-border);
}

.glass-card {
  background: var(--glass-bg);
  backdrop-filter: blur(var(--glass-blur));
  -webkit-backdrop-filter: blur(var(--glass-blur));
  border: 1px solid var(--glass-border);
  border-radius: 0.75rem;
}
```

#### Usage
```tsx
<div className="glass p-6">
  {/* Content with glass background */}
</div>
```

**When to use:**
- Overlays
- Modals
- Floating panels
- Subtle depth effects

### Cards

Standard card pattern for content containers.

```tsx
<Card className="
  overflow-hidden 
  border-4 border-primary/30 
  shadow-[8px_8px_0_rgba(0,0,0,0.2)] 
  hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] 
  transition-all
">
  {/* Card content */}
</Card>
```

**Variants:**

**Emphasized card (full border):**
```tsx
border-4 border-primary
```

**Dashed border (create actions):**
```tsx
border-4 border-dashed border-primary/50
```

**Destructive emphasis:**
```tsx
border-4 border-destructive
```

### Tables

Retro-styled data tables with heavy borders.

```tsx
<div className="
  bg-card rounded-xl overflow-hidden 
  border-4 border-primary 
  shadow-[8px_8px_0_rgba(0,0,0,0.6)]
">
  <table className="w-full">
    <thead className="bg-muted/50 border-b-4 border-primary">
      <tr>
        <th className="text-left px-6 py-4 font-bold vcr-text">
          Column Header
        </th>
      </tr>
    </thead>
    <tbody>
      <tr className="border-b border-border hover:bg-muted/30 transition-colors">
        <td className="px-6 py-4">
          Cell content
        </td>
      </tr>
    </tbody>
  </table>
</div>
```

**Key elements:**
- Container has heavy shadow: `shadow-[8px_8px_0_rgba(0,0,0,0.6)]`
- Header has bottom border: `border-b-4 border-primary`
- Header text uses `.vcr-text`
- Rows have hover state: `hover:bg-muted/30`

### Status Badges

Inline status indicators with dot and text.

```tsx
<span className="
  inline-flex items-center gap-2 
  px-3 py-1 rounded-full 
  bg-primary/20 text-primary 
  border-2 border-primary 
  font-bold text-sm
">
  <span className="w-2 h-2 bg-primary rounded-full animate-pulse" />
  READY
</span>
```

**Variants:**

**Live indicator:**
```tsx
bg-destructive/20 text-destructive border-2 border-destructive
```

**Offline/inactive:**
```tsx
bg-muted text-muted-foreground border-2 border-border
```

**Success:**
```tsx
bg-accent/20 text-accent border-2 border-accent
```

---

## Animations & Transitions

### Live Pulse Animation

For live indicators and attention-grabbing elements.

```css
.live-pulse {
  animation: live-pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes live-pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
```

**Usage:**
```tsx
<span className="w-3 h-3 bg-destructive rounded-full live-pulse" />
```

### CRT Scanline Animation

Vertical scanning effect for CRT screens.

```css
@keyframes scanline {
  0% {
    transform: translateY(-100%);
  }
  100% {
    transform: translateY(100%);
  }
}
/* Applied automatically to .crt-screen::after */
/* Duration: 8s linear infinite */
```

### CRT Flicker Animation

Subtle flicker effect for authenticity.

```css
@keyframes crt-flicker {
  0%, 100% {
    opacity: 0.97;
  }
  50% {
    opacity: 1;
  }
}
/* Applied automatically to .crt-screen::before */
/* Duration: 0.15s infinite */
```

### Standard Transitions

#### Smooth Transition (Custom Utility)
```css
.transition-smooth {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
```

**Usage:**
```tsx
<div className="transition-smooth hover:scale-[1.02]">
  {/* Content */}
</div>
```

#### Standard Tailwind Transitions
```tsx
transition-all        /* All properties */
transition-colors     /* Colors only */
transition-transform  /* Transform only */
transition-opacity    /* Opacity only */
```

**Timing:**
- Use default duration for most interactions
- Button presses: `transition: all 0.1s ease` (faster, more responsive)
- Page elements: `transition-smooth` or `transition-all` (300ms)

---

## Layout Components

### RetroHeaderLayout

The primary layout wrapper providing consistent header and navigation.

**File:** `/web/components/layout/retro-header-layout.tsx`

```tsx
import { RetroHeaderLayout } from "@/components/layout/retro-header-layout"

export default function Page() {
  return (
    <RetroHeaderLayout>
      {/* Page content */}
    </RetroHeaderLayout>
  )
}
```

**Features:**
- Sticky header: `sticky top-0 z-50`
- Border: `border-b-4 border-primary`
- Backdrop blur for depth: `backdrop-blur-sm`
- Radio icon branding
- "Virtual TV" title with `.vcr-text`
- Navigation buttons (CHANNELS, LIBRARY)
- Theme toggle
- Settings button

**Structure:**
```tsx
<div className="min-h-screen bg-background">
  {/* Header */}
  <header className="
    border-b-4 border-primary 
    backdrop-blur-sm 
    sticky top-0 z-50 
    bg-card shadow-lg
  ">
    <div className="container mx-auto px-6 py-4">
      {/* Branding, navigation, actions */}
    </div>
  </header>

  {/* Main Content */}
  <main className="container mx-auto px-6 py-8">
    {children}
  </main>
</div>
```

**When to use:**
- All main application pages
- Standard content pages
- Dashboard views

---

## State Patterns

### Loading States

Use Skeleton components with consistent styling.

```tsx
import { Skeleton } from "@/components/ui/skeleton"

{isLoading && (
  <Card className="border-4 border-primary/30">
    <Skeleton className="aspect-video w-full" />
    <div className="p-6 space-y-3">
      <Skeleton className="h-6 w-3/4" />
      <Skeleton className="h-4 w-full" />
    </div>
  </Card>
)}
```

**Key principles:**
- Match the shape of final content
- Use same border styling as content
- Show multiple skeletons for lists/grids
- Maintain layout stability (no shift when loaded)

### Error States

Prominent error messaging with destructive styling.

```tsx
{isError && (
  <div className="
    bg-card rounded-xl p-8 
    border-4 border-destructive 
    shadow-[8px_8px_0_rgba(0,0,0,0.6)] 
    text-center
  ">
    <p className="text-destructive font-bold text-lg vcr-text">
      Failed to load channels
    </p>
    <p className="text-muted-foreground mt-2">
      Please try again later
    </p>
  </div>
)}
```

**Key elements:**
- Heavy border: `border-4 border-destructive`
- Heavy shadow: `shadow-[8px_8px_0_rgba(0,0,0,0.6)]`
- VCR text for error message
- Optional retry button
- Centered content

### Empty States

Encouraging empty states with clear calls to action.

```tsx
{items.length === 0 && (
  <div className="
    bg-card rounded-xl p-12 
    border-4 border-primary/20 
    shadow-[8px_8px_0_rgba(0,0,0,0.2)] 
    text-center
  ">
    <p className="text-muted-foreground font-mono text-lg">
      No channels found
    </p>
    <Link href="/channels/new">
      <button className="
        mt-4 retro-button 
        bg-primary text-primary-foreground 
        hover:bg-primary/80 
        px-6 py-3 rounded-lg font-bold 
        border-2 border-primary-foreground/20 
        shadow-[6px_6px_0_rgba(0,0,0,0.2)] 
        hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] 
        transition-all
      ">
        CREATE YOUR FIRST CHANNEL
      </button>
    </Link>
  </div>
)}
```

**For "create" cards:**
```tsx
<Card className="
  border-4 border-dashed border-primary/50 
  shadow-[8px_8px_0_rgba(0,0,0,0.2)] 
  hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] 
  bg-card/50
">
  {/* Dashed border indicates "add new" action */}
</Card>
```

---

## Usage Guidelines

### When to Use Retro Buttons

**Always use `.retro-button` for:**
- Primary action buttons
- Navigation elements
- Form submissions
- All interactive buttons

**Always uppercase:**
```tsx
CREATE CHANNEL  ✓
EDIT           ✓
DELETE         ✓
Save Changes   ✗ (should be SAVE CHANGES)
```

### When to Use CRT Effects

**Apply `.crt-screen` to:**
- Video thumbnails
- Media preview images
- Channel display areas
- Content viewing areas

**Do NOT apply to:**
- UI chrome (buttons, headers, navigation)
- Form elements
- Text content
- Background surfaces

### Border Weight Selection

| Element Type | Border Weight | Example |
|--------------|---------------|---------|
| Buttons | `border-2` | All retro buttons |
| Inputs | `border-2` or `border-4` | Forms, search boxes |
| Status badges | `border-2` | Live, Ready, Offline |
| Cards | `border-4` | Content cards, panels |
| Tables | `border-4` | Data tables |
| Major containers | `border-4` | Page sections |
| Headers | `border-b-4` | Header bottom border |

### Color Semantic Meanings

| Token | Meaning | Usage |
|-------|---------|-------|
| `primary` | Brand, main actions | CTAs, branding, focus states |
| `secondary` | Content backgrounds | Card backgrounds, sections |
| `accent` | Highlights, secondary actions | Edit buttons, highlights, success |
| `destructive` | Errors, warnings, delete | Error messages, delete buttons, alerts |
| `muted` | Disabled, secondary info | Disabled buttons, secondary text |

### Typography Hierarchy

**Page structure:**
```tsx
{/* Page title */}
<h1 className="text-4xl font-bold vcr-text">Page Title</h1>

{/* Page description */}
<p className="text-lg text-muted-foreground font-mono">
  Description text
</p>

{/* Section heading */}
<h2 className="text-3xl font-bold vcr-text">Section Title</h2>

{/* Body text */}
<p className="font-mono">Body content...</p>

{/* Label/caption */}
<span className="text-sm text-muted-foreground font-mono">
  Metadata
</span>
```

**Button text:**
```tsx
{/* Always uppercase, always bold */}
<button className="retro-button font-bold">
  CREATE CHANNEL
</button>
```

---

## Theme System

### Implementation

Themes are applied via CSS classes on the `<html>` element:

```tsx
<html lang="en" className="dark sunset-vhs">
  {/* Application */}
</html>
```

### Theme Toggle Component

Use the `ThemeToggle` component for theme switching:

```tsx
import { ThemeToggle } from "@/components/theme-toggle"

<ThemeToggle />
```

**Implementation:**
```tsx
const themes = [
  { id: "modern", name: "Modern Clean", classes: "dark" },
  { id: "neon-arcade", name: "Neon Arcade", classes: "dark retro neon-arcade" },
  { id: "sunset-vhs", name: "Sunset VHS", classes: "dark retro sunset-vhs" },
  { id: "retro-gaming", name: "Retro Gaming", classes: "dark retro retro-gaming" },
  { id: "miami-vice", name: "Miami Vice", classes: "dark retro miami-vice" },
]
```

### Available Themes

#### 1. Modern Clean (Dark)
**Class:** `dark`

A sophisticated dark theme inspired by modern streaming services like Netflix and Disney+. Deep purples and blues with high contrast.

**Best for:** Default theme, professional appearance

#### 2. Retro
**Class:** `dark retro`

Warm 80s/90s Saturday morning cartoon aesthetic with beige backgrounds, teal primaries, and hot pink accents.

**Best for:** Maximum nostalgia, playful interfaces

#### 3. Neon Arcade
**Class:** `dark retro neon-arcade`

Electric purple and magenta with cyan accents. Inspired by arcade machines and synthwave aesthetics.

**Best for:** Bold, energetic interfaces

#### 4. Sunset VHS
**Class:** `dark retro sunset-vhs`

Coral pink and deep purple with warm amber accents. VHS tape aesthetic with sunset colors.

**Best for:** Warm, nostalgic atmosphere

#### 5. Retro Gaming
**Class:** `dark retro retro-gaming`

Lime green primary with hot magenta and orange. Classic gaming console colors.

**Best for:** Vibrant, game-like interfaces

#### 6. Miami Vice
**Class:** `dark retro miami-vice`

Hot pink and turquoise with purple. 80s Miami aesthetic.

**Best for:** Bold, tropical vibe

### Theme Customization

All themes use the same semantic tokens. To create a new theme:

1. Define color values in `globals.css`:
```css
.my-theme {
  --background: oklch(...);
  --foreground: oklch(...);
  --primary: oklch(...);
  /* ... other tokens */
}
```

2. Add to theme toggle:
```tsx
{ id: "my-theme", name: "My Theme", classes: "dark retro my-theme" }
```

3. Use OKLCH color space for consistency

### Theme Testing

Test themes by:
1. Checking all semantic colors are defined
2. Verifying contrast ratios for accessibility
3. Testing on various components (buttons, cards, text)
4. Ensuring CRT effects work with theme colors

---

## Quick Reference

### Common Patterns

**Primary button:**
```tsx
<button className="retro-button bg-primary text-primary-foreground hover:bg-primary/80 px-6 py-3 rounded-lg font-bold border-2 border-primary-foreground/20 shadow-[6px_6px_0_rgba(0,0,0,0.2)] hover:shadow-[3px_3px_0_rgba(0,0,0,0.2)] transition-all">
  BUTTON TEXT
</button>
```

**Card:**
```tsx
<Card className="border-4 border-primary/30 shadow-[8px_8px_0_rgba(0,0,0,0.2)] hover:shadow-[4px_4px_0_rgba(0,0,0,0.2)] transition-all">
  {/* Content */}
</Card>
```

**CRT thumbnail:**
```tsx
<div className="crt-screen aspect-video bg-muted">
  <img src={thumbnail} alt={title} className="w-full h-full object-cover" />
</div>
```

**Page title:**
```tsx
<h1 className="text-4xl font-bold text-foreground vcr-text">Page Title</h1>
```

**Status badge:**
```tsx
<span className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-primary/20 text-primary border-2 border-primary font-bold text-sm">
  <span className="w-2 h-2 bg-primary rounded-full" />
  STATUS
</span>
```

---

## Resources

- **Color System:** `/web/app/globals.css` (lines 1-442)
- **Layout:** `/web/components/layout/retro-header-layout.tsx`
- **Theme Toggle:** `/web/components/theme-toggle.tsx`
- **Example Pages:**
  - Home: `/web/app/page.tsx`
  - Channels: `/web/app/channels/page.tsx`
  - Library: `/web/app/library/page.tsx`

---

**Design System Version 1.0** | Virtual TV | October 2025

