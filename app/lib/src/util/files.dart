import 'dart:convert';
import 'dart:io';

import 'package:file_selector/file_selector.dart';

const csvTypes = [
  XTypeGroup(label: 'CSV', extensions: ['csv']),
];
const jsonTypes = [
  XTypeGroup(label: 'JSON', extensions: ['json']),
];

/// Opens a text file chosen by the user; null if they cancel.
Future<({String name, String text})?> pickTextFile(List<XTypeGroup> types) async {
  final file = await openFile(acceptedTypeGroups: types);
  if (file == null) return null;
  final bytes = await file.readAsBytes();
  return (name: file.name, text: utf8.decode(bytes, allowMalformed: true));
}

/// Saves text where the user chooses; returns the path, or null if they cancel.
Future<String?> saveTextFile(String suggestedName, String text, List<XTypeGroup> types) async {
  final location = await getSaveLocation(suggestedName: suggestedName, acceptedTypeGroups: types);
  if (location == null) return null;
  await File(location.path).writeAsString(text);
  return location.path;
}

String baseName(String fileName) {
  final dot = fileName.lastIndexOf('.');
  return dot > 0 ? fileName.substring(0, dot) : fileName;
}

String prettyJson(Object value) => '${const JsonEncoder.withIndent('  ').convert(value)}\n';
