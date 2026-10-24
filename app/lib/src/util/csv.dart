/// Minimal RFC 4180 CSV parser, used for previews and row counts only; the
/// engine does the real parsing.
List<List<String>> parseCsv(String text, {int? maxRows}) {
  final rows = <List<String>>[];
  var row = <String>[];
  final field = StringBuffer();
  var inQuotes = false;
  var i = text.startsWith('\uFEFF') ? 1 : 0;

  void endRow() {
    row.add(field.toString());
    field.clear();
    rows.add(row);
    row = <String>[];
  }

  for (; i < text.length; i++) {
    final c = text[i];
    if (inQuotes) {
      if (c == '"') {
        if (i + 1 < text.length && text[i + 1] == '"') {
          field.write('"');
          i++;
        } else {
          inQuotes = false;
        }
      } else {
        field.write(c);
      }
    } else if (c == '"') {
      inQuotes = true;
    } else if (c == ',') {
      row.add(field.toString());
      field.clear();
    } else if (c == '\n' || c == '\r') {
      if (c == '\r' && i + 1 < text.length && text[i + 1] == '\n') i++;
      endRow();
      if (maxRows != null && rows.length >= maxRows) return rows;
    } else {
      field.write(c);
    }
  }
  if (field.isNotEmpty || row.isNotEmpty) endRow();

  return rows;
}
