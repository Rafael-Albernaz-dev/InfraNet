import { DataTable } from '../../shared/ui/DataTable';
export function ImportTemplateFields() {
  return (
    <section className="import-card">
      <h2>Campos do modelo</h2>
      <p className="muted">
        Não há campo Nome: o nome vem da identidade lida de cada equipamento. No
        JSON, uma lista de objetos com estes campos.
      </p>
      <DataTable plain>
        <thead>
          <tr>
            <th>Campo</th>
            <th>Obrigatório</th>
            <th>Exemplo</th>
          </tr>
        </thead>
        <tbody>
          {[
            ['IP', 'Sim', '192.0.2.31'],
            ['Porta', 'Não, padrão 22', '22'],
            ['Papel', 'Sim', 'AP'],
            ['Grupo', 'Não', 'Região Norte › Torre Norte'],
            ['Credencial', 'Não', 'leitura-radio'],
          ].map(([name, required, example]) => (
            <tr key={name}>
              <td>{name}</td>
              <td>{required}</td>
              <td>{example}</td>
            </tr>
          ))}
        </tbody>
      </DataTable>
    </section>
  );
}
