import Component from '@ember/component';
import { computed } from '@ember/object';

/**
 * @module JsonViewer
 * `JsonViewer` renders JSON with syntax highlighting.
 *
 * @example
 * {{json-viewer json=someJson}}
 *
 * @param json {Object} - the JSON to render
 *
 */
export default Component.extend({
  classNames: ['json-viewer'],

  json: null,
  jsonStr: computed('json', function() {
    return JSON.stringify(this.json, null, 2);
  }),
});
