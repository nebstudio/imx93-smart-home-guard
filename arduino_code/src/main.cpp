#include <Arduino.h>
#include <Servo.h>
#include <Wire.h>
#include <LiquidCrystal_I2C.h>

#define PIN_BUZZER          2
#define PIN_ULTRASONIC_TRIG 3
#define PIN_ULTRASONIC_ECHO 4
#define PIN_FAN_PWM         5
#define PIN_FAN_DIR         6
#define PIN_LIGHT_RED       7
#define PIN_LIGHT_YELLOW    8
#define PIN_LIGHT_GREEN     9
#define PIN_SERVO_1         10
#define PIN_SERVO_2         11
#define PIN_SERVO_3         12
#define PIN_IR_OBSTACLE     13

#define PIN_SMOKE_SENSOR    A1
#define PIN_FLAME_SENSOR    A2

LiquidCrystal_I2C *lcd = nullptr;
bool lcdReady = false;

const uint8_t LCD_CANDIDATE_ADDRESSES[] = {0x27, 0x3F};
const uint8_t LCD_COLS = 16;
const uint8_t LCD_ROWS = 2;

#define MAX_LCD_TEXT_LEN 48
char lcdLineText[LCD_ROWS][MAX_LCD_TEXT_LEN + 1];
uint8_t lcdLineLen[LCD_ROWS] = {0, 0};
uint16_t lcdScrollPos[LCD_ROWS] = {0, 0};
unsigned long lcdLastScrollMs[LCD_ROWS] = {0, 0};
const unsigned long LCD_SCROLL_STEP_MS = 400;

void renderLcdLine(uint8_t line) {
  if (!lcdReady) return;

  char window[LCD_COLS + 1];
  uint8_t len = lcdLineLen[line];

  if (len <= LCD_COLS) {
    uint8_t i = 0;
    for (; i < len; i++) window[i] = lcdLineText[line][i];
    for (; i < LCD_COLS; i++) window[i] = ' ';
  } else {

    uint16_t virtualLen = (uint16_t)len + LCD_COLS;
    uint16_t pos = lcdScrollPos[line];
    for (uint8_t i = 0; i < LCD_COLS; i++) {
      uint16_t idx = (pos + i) % virtualLen;
      window[i] = (idx < len) ? lcdLineText[line][idx] : ' ';
    }
  }
  window[LCD_COLS] = '\0';

  lcd->setCursor(0, line);
  lcd->print(window);
}

void updateLcdScroll() {
  if (!lcdReady) return;
  unsigned long now = millis();
  for (uint8_t line = 0; line < LCD_ROWS; line++) {
    if (lcdLineLen[line] <= LCD_COLS) continue;
    if (now - lcdLastScrollMs[line] < LCD_SCROLL_STEP_MS) continue;
    lcdLastScrollMs[line] = now;
    uint16_t virtualLen = (uint16_t)lcdLineLen[line] + LCD_COLS;
    lcdScrollPos[line] = (lcdScrollPos[line] + 1) % virtualLen;
    renderLcdLine(line);
  }
}

uint8_t probeLcdAddress() {
  for (uint8_t i = 0; i < sizeof(LCD_CANDIDATE_ADDRESSES); i++) {
    uint8_t addr = LCD_CANDIDATE_ADDRESSES[i];
    Wire.beginTransmission(addr);
    if (Wire.endTransmission() == 0) {
      return addr;
    }
  }
  return 0;
}

byte lcdBigSmiley0[8] = {0b00000, 0b00000, 0b00110, 0b00110, 0b00110, 0b00000, 0b00000, 0b00000};
byte lcdBigSmiley1[8] = {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigSmiley2[8] = {0b00000, 0b00000, 0b11000, 0b11000, 0b11000, 0b00000, 0b00000, 0b00000};
byte lcdBigSmiley3[8] = {0b10000, 0b11000, 0b01100, 0b00111, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigSmiley4[8] = {0b00000, 0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigSmiley5[8] = {0b00001, 0b00011, 0b01100, 0b11000, 0b00000, 0b00000, 0b00000, 0b00000};

byte lcdBigFrownie0[8] = {0b00000, 0b00000, 0b00110, 0b00110, 0b00110, 0b00000, 0b00000, 0b00000};
byte lcdBigFrownie1[8] = {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigFrownie2[8] = {0b00000, 0b00000, 0b11000, 0b11000, 0b11000, 0b00000, 0b00000, 0b00000};
byte lcdBigFrownie3[8] = {0b00111, 0b01100, 0b11000, 0b10000, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigFrownie4[8] = {0b11111, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigFrownie5[8] = {0b11000, 0b01100, 0b00011, 0b00001, 0b00000, 0b00000, 0b00000, 0b00000};

byte lcdBigSurprised0[8] = {0b00000, 0b00000, 0b00110, 0b01001, 0b01001, 0b00110, 0b00000, 0b00000};
byte lcdBigSurprised1[8] = {0b00000, 0b00000, 0b00000, 0b00001, 0b00001, 0b00000, 0b00000, 0b00000};
byte lcdBigSurprised2[8] = {0b00000, 0b00000, 0b11000, 0b00100, 0b00100, 0b11000, 0b00000, 0b00000};
byte lcdBigSurprised3[8] = {0b00000, 0b00000, 0b00001, 0b00001, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigSurprised4[8] = {0b00000, 0b11110, 0b00001, 0b00001, 0b11110, 0b00000, 0b00000, 0b00000};
byte lcdBigSurprised5[8] = {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};

byte lcdBigNeutral0[8] = {0b00000, 0b00000, 0b00000, 0b00110, 0b00110, 0b00000, 0b00000, 0b00000};
byte lcdBigNeutral1[8] = {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigNeutral2[8] = {0b00000, 0b00000, 0b00000, 0b11000, 0b11000, 0b00000, 0b00000, 0b00000};
byte lcdBigNeutral3[8] = {0b00000, 0b00000, 0b00001, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigNeutral4[8] = {0b00000, 0b00000, 0b11111, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};
byte lcdBigNeutral5[8] = {0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000, 0b00000};

#define LCD_EMOJI_SMILEY    0
#define LCD_EMOJI_FROWNIE   1
#define LCD_EMOJI_SURPRISED 2
#define LCD_EMOJI_NEUTRAL   3
#define LCD_EMOJI_COUNT     4

#define LCD_BIG_EMOJI_SLOT_COUNT 6

#define LCD_BIG_EMOJI_START_COL 6

void initLcd() {
  uint8_t addr = probeLcdAddress();
  if (addr == 0) {
    lcdReady = false;
    return;
  }
  lcd = new LiquidCrystal_I2C(addr, LCD_COLS, LCD_ROWS);
  lcd->init();
  lcd->backlight();
  lcd->clear();
  lcdReady = true;
}

bool emojiBlockSet(uint8_t emojiIndex, byte *out[LCD_BIG_EMOJI_SLOT_COUNT]) {
  switch (emojiIndex) {
    case LCD_EMOJI_SMILEY:
      out[0] = lcdBigSmiley0; out[1] = lcdBigSmiley1; out[2] = lcdBigSmiley2;
      out[3] = lcdBigSmiley3; out[4] = lcdBigSmiley4; out[5] = lcdBigSmiley5;
      return true;
    case LCD_EMOJI_FROWNIE:
      out[0] = lcdBigFrownie0; out[1] = lcdBigFrownie1; out[2] = lcdBigFrownie2;
      out[3] = lcdBigFrownie3; out[4] = lcdBigFrownie4; out[5] = lcdBigFrownie5;
      return true;
    case LCD_EMOJI_SURPRISED:
      out[0] = lcdBigSurprised0; out[1] = lcdBigSurprised1; out[2] = lcdBigSurprised2;
      out[3] = lcdBigSurprised3; out[4] = lcdBigSurprised4; out[5] = lcdBigSurprised5;
      return true;
    case LCD_EMOJI_NEUTRAL:
      out[0] = lcdBigNeutral0; out[1] = lcdBigNeutral1; out[2] = lcdBigNeutral2;
      out[3] = lcdBigNeutral3; out[4] = lcdBigNeutral4; out[5] = lcdBigNeutral5;
      return true;
    default:
      return false;
  }
}

bool lcdShowBigEmoji(uint8_t emojiIndex) {
  if (!lcdReady) return false;

  byte *blocks[LCD_BIG_EMOJI_SLOT_COUNT];
  if (!emojiBlockSet(emojiIndex, blocks)) {
    return false;
  }
  for (uint8_t i = 0; i < LCD_BIG_EMOJI_SLOT_COUNT; i++) {
    lcd->createChar(i, blocks[i]);
  }

  lcd->clear();
  for (uint8_t line = 0; line < LCD_ROWS; line++) {
    lcdLineLen[line] = 0;
  }
  for (uint8_t row = 0; row < 2; row++) {
    lcd->setCursor(LCD_BIG_EMOJI_START_COL, row);
    for (uint8_t col = 0; col < 3; col++) {
      lcd->write(row * 3 + col);
    }
  }
  return true;
}

void lcdShowLine(uint8_t line, const char *text) {
  if (!lcdReady || line >= LCD_ROWS) return;

  uint8_t i = 0;
  for (; i < MAX_LCD_TEXT_LEN && text[i] != '\0'; i++) {
    lcdLineText[line][i] = text[i];
  }
  lcdLineText[line][i] = '\0';
  lcdLineLen[line] = i;
  lcdScrollPos[line] = 0;
  lcdLastScrollMs[line] = millis();

  renderLcdLine(line);
}

Servo servo1, servo2, servo3;
bool servo1Attached = false, servo2Attached = false, servo3Attached = false;

unsigned long servoMoveDelayMs(int angle) {
  return 400;
}

const uint8_t CMD_BUF_SIZE = 32;
char cmdBuf[CMD_BUF_SIZE];
uint8_t cmdLen = 0;

long measureDistanceCm() {
  digitalWrite(PIN_ULTRASONIC_TRIG, LOW);
  delayMicroseconds(2);
  digitalWrite(PIN_ULTRASONIC_TRIG, HIGH);
  delayMicroseconds(10);
  digitalWrite(PIN_ULTRASONIC_TRIG, LOW);

  unsigned long duration = pulseIn(PIN_ULTRASONIC_ECHO, HIGH, 30000UL);
  if (duration == 0) {
    return -1;
  }
  return (long)(duration / 58.0);
}

void fanControl(int dir, int speed) {
  speed = constrain(speed, 0, 255);
  digitalWrite(PIN_FAN_DIR, dir == 0 ? LOW : HIGH);
  analogWrite(PIN_FAN_PWM, speed);
}

Servo *servoByIndex(int index) {
  switch (index) {
    case 1: return &servo1;
    case 2: return &servo2;
    case 3: return &servo3;
    default: return nullptr;
  }
}

bool *servoAttachedFlag(int index) {
  switch (index) {
    case 1: return &servo1Attached;
    case 2: return &servo2Attached;
    case 3: return &servo3Attached;
    default: return nullptr;
  }
}

int servoPinByIndex(int index) {
  switch (index) {
    case 1: return PIN_SERVO_1;
    case 2: return PIN_SERVO_2;
    case 3: return PIN_SERVO_3;
    default: return -1;
  }
}

bool isValidDigitalPin(int pin) {
  return pin >= 2 && pin <= 13;
}

bool isValidAnalogPin(int pin) {
  return pin >= 0 && pin <= 5;
}

int analogPinFromIndex(int idx) {
  switch (idx) {
    case 0: return A0;
    case 1: return A1;
    case 2: return A2;
    case 3: return A3;
    case 4: return A4;
    case 5: return A5;
    default: return -1;
  }
}

void replyError(const char *raw) {
  Serial.print("ERR,");
  Serial.println(raw);
}

void handleCommand(char *line) {
  size_t len = strlen(line);
  if (len == 0) return;

  char original[CMD_BUF_SIZE];
  strncpy(original, line, CMD_BUF_SIZE - 1);
  original[CMD_BUF_SIZE - 1] = '\0';

  char type = line[0];
  bool isQuery = (line[len - 1] == '?');

  if (strncmp(line, "PING?", 5) == 0) {
    Serial.println("PONG");
    return;
  }

  if (type == 'U' && isQuery) {
    long dist = measureDistanceCm();
    Serial.print("U,");
    Serial.println(dist);
    return;
  }

  if (strncmp(line, "LS?", 3) == 0) {
    Serial.print("LS,");
    Serial.println(lcdReady ? 1 : 0);
    return;
  }

  if (strncmp(line, "LC", 2) == 0 && len == 2) {
    if (!lcdReady) { replyError(original); return; }
    lcd->clear();

    for (uint8_t i = 0; i < LCD_ROWS; i++) {
      lcdLineLen[i] = 0;
      lcdLineText[i][0] = '\0';
      lcdScrollPos[i] = 0;
    }
    Serial.println("OK");
    return;
  }

  if (strncmp(line, "LE", 2) == 0 && len == 3) {
    if (!lcdReady) { replyError(original); return; }
    int emojiIndex = line[2] - '0';
    if (emojiIndex < 0 || emojiIndex >= LCD_EMOJI_COUNT || !lcdShowBigEmoji((uint8_t)emojiIndex)) {
      replyError(original);
      return;
    }
    Serial.println("OK");
    return;
  }

  if (type == 'L' && len >= 2 && line[1] != 'C' && line[1] != 'S' && line[1] != 'E') {
    char *comma = strchr(line, ',');
    if (comma == nullptr) { replyError(original); return; }
    *comma = '\0';
    int lineIdx = atoi(line + 1);
    if (lineIdx < 0 || lineIdx >= LCD_ROWS) { replyError(original); return; }
    if (!lcdReady) { replyError(original); return; }
    lcdShowLine((uint8_t)lineIdx, comma + 1);
    Serial.println("OK");
    return;
  }

  if (type == 'A' && isQuery) {
    int pinIdx = atoi(line + 1);
    if (!isValidAnalogPin(pinIdx)) { replyError(original); return; }
    int pin = analogPinFromIndex(pinIdx);
    int val = analogRead(pin);
    Serial.print("A");
    Serial.print(pinIdx);
    Serial.print(",");
    Serial.println(val);
    return;
  }

  if (type == 'D' && isQuery) {
    int pin = atoi(line + 1);
    if (!isValidDigitalPin(pin)) { replyError(original); return; }
    pinMode(pin, INPUT);
    int val = digitalRead(pin);
    Serial.print("D");
    Serial.print(pin);
    Serial.print(",");
    Serial.println(val);
    return;
  }

  char *comma = strchr(line, ',');
  if (comma == nullptr) { replyError(original); return; }
  *comma = '\0';
  int p1 = atoi(line + 1);
  int p2 = atoi(comma + 1);

  switch (type) {
    case 'D': {
      if (!isValidDigitalPin(p1)) { replyError(original); return; }
      pinMode(p1, OUTPUT);
      digitalWrite(p1, p2 != 0 ? HIGH : LOW);
      break;
    }
    case 'P': {
      if (!isValidDigitalPin(p1)) { replyError(original); return; }
      pinMode(p1, OUTPUT);
      analogWrite(p1, constrain(p2, 0, 255));
      break;
    }
    case 'S': {

      Servo *sv = servoByIndex(p1);
      bool *attachedFlag = servoAttachedFlag(p1);
      int pin = servoPinByIndex(p1);
      if (sv == nullptr || attachedFlag == nullptr) { replyError(original); return; }

      int angle = constrain(p2, 0, 180);
      if (!(*attachedFlag)) {
        sv->attach(pin);
        *attachedFlag = true;
      }
      sv->write(angle);
      delay(servoMoveDelayMs(angle));
      sv->detach();
      *attachedFlag = false;
      break;
    }
    case 'B': {
      if (p1 <= 0) {
        noTone(PIN_BUZZER);
      } else {
        tone(PIN_BUZZER, p1, p2);
      }
      break;
    }
    case 'M': {
      fanControl(p1, p2);
      break;
    }
    default:
      replyError(original);
      return;
  }

  Serial.println("OK");
}

void setup() {
  Serial.begin(115200);

  Wire.begin();
  initLcd();

  pinMode(PIN_ULTRASONIC_TRIG, OUTPUT);
  pinMode(PIN_ULTRASONIC_ECHO, INPUT);
  pinMode(PIN_BUZZER, OUTPUT);
  pinMode(PIN_FAN_PWM, OUTPUT);
  pinMode(PIN_FAN_DIR, OUTPUT);
  pinMode(PIN_LIGHT_RED, OUTPUT);
  pinMode(PIN_LIGHT_YELLOW, OUTPUT);
  pinMode(PIN_LIGHT_GREEN, OUTPUT);
  pinMode(PIN_IR_OBSTACLE, INPUT);

  digitalWrite(PIN_FAN_PWM, LOW);
  digitalWrite(PIN_FAN_DIR, LOW);

  servo1.attach(PIN_SERVO_1);
  servo1.write(90);
  delay(400);
  servo1.detach();

  servo2.attach(PIN_SERVO_2);
  servo2.write(90);
  delay(400);
  servo2.detach();

  servo3.attach(PIN_SERVO_3);
  servo3.write(90);
  delay(400);
  servo3.detach();

  digitalWrite(PIN_LIGHT_GREEN, HIGH);
  digitalWrite(PIN_LIGHT_RED, LOW);
  digitalWrite(PIN_LIGHT_YELLOW, LOW);

  if (lcdReady) {
    lcdShowBigEmoji(LCD_EMOJI_SMILEY);
  }

  Serial.println("READY");
}

void loop() {

  updateLcdScroll();

  while (Serial.available() > 0) {
    char c = (char)Serial.read();
    if (c == '\n' || c == '\r') {
      if (cmdLen > 0) {
        cmdBuf[cmdLen] = '\0';
        handleCommand(cmdBuf);
        cmdLen = 0;
      }
    } else if (cmdLen < CMD_BUF_SIZE - 1) {
      cmdBuf[cmdLen++] = c;
    }

  }
}
