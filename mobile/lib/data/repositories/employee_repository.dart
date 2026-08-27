import '../../core/api_client.dart';

class EmployeeSummary {
  EmployeeSummary({required this.id, required this.name, required this.employeeNumber});

  final String id;
  final String name;
  final String employeeNumber;

  factory EmployeeSummary.fromApi(Map<String, dynamic> json) => EmployeeSummary(
        id: json['id'] as String,
        name: json['name'] as String,
        employeeNumber: json['employee_number'] as String,
      );
}

/// Read-only employee lookup used by the enrollment screen's search box.
/// JWT-gated like every other dashboard read (see AuthRepository).
class EmployeeRepository {
  EmployeeRepository(this._api);

  final ApiClient _api;

  Future<List<EmployeeSummary>> search(String query) async {
    final data = await _api.get('/employees', query: {
      if (query.isNotEmpty) 'search': query,
      'page_size': '20',
    });
    final items = data['items'] as List;
    return items.map((e) => EmployeeSummary.fromApi(e as Map<String, dynamic>)).toList();
  }
}
