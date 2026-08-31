import ace from 'ace-builds'
import 'ace-builds/src-noconflict/mode-text'

let registered = false

export function registerIotLogMode() {
  if (registered) {
    return
  }
  registered = true

  ace.define(
    'ace/mode/iot_log_highlight_rules',
    ['require', 'exports', 'module', 'ace/lib/oop', 'ace/mode/text_highlight_rules'],
    function (require, exports) {
      const oop = require('ace/lib/oop')
      const TextHighlightRules = require('ace/mode/text_highlight_rules').TextHighlightRules

      function IotLogHighlightRules() {
        this.$rules = {
          start: [
            {
              token: 'log-time',
              regex: /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?/
            },
            { token: 'log-level-debug', regex: /\[DEBUG\]/ },
            { token: 'log-level-info', regex: /\[INFO\]/ },
            { token: 'log-level-warn', regex: /\[WARN\]/ },
            { token: 'log-level-error', regex: /\[ERROR\]/ },
            {
              token: 'log-device',
              regex: /(?<=\[(?:DEBUG|INFO|WARN|ERROR)\] )\[[^\]]+\]/
            },
            { token: 'log-json-key', regex: /"(?:\\.|[^"\\])*"\s*(?=:)/ },
            { token: 'string', regex: /"(?:\\.|[^"\\])*"/ },
            { token: 'constant.language', regex: /\b(?:true|false|null)\b/ },
            { token: 'constant.numeric', regex: /-?\b\d+(?:\.\d+)?(?:[eE][+-]?\d+)?\b/ }
          ]
        }
        this.normalizeRules()
      }

      oop.inherits(IotLogHighlightRules, TextHighlightRules)
      exports.IotLogHighlightRules = IotLogHighlightRules
    }
  )

  ace.define(
    'ace/mode/iot_log',
    [
      'require',
      'exports',
      'module',
      'ace/lib/oop',
      'ace/mode/text',
      'ace/mode/iot_log_highlight_rules'
    ],
    function (require, exports) {
      const oop = require('ace/lib/oop')
      const TextMode = require('ace/mode/text').Mode
      const IotLogHighlightRules = require('ace/mode/iot_log_highlight_rules').IotLogHighlightRules

      function Mode() {
        this.HighlightRules = IotLogHighlightRules
        this.$id = 'ace/mode/iot_log'
      }

      oop.inherits(Mode, TextMode)
      exports.Mode = Mode
    }
  )
}

export function createIotLogMode() {
  registerIotLogMode()
  return new (ace.require('ace/mode/iot_log').Mode)()
}

export function escapeRegExp(text) {
  return String(text).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
