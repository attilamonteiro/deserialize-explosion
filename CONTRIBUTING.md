# Contribuindo

Obrigado por considerar contribuir com este repositório. Algumas recomendações rápidas:

- Abra uma issue antes de mudanças grandes para alinhar o escopo.
- Faça um fork e crie uma branch com nome descritivo: `fix/perpage-cache` ou `feat/cleancpf`.
- Escreva commits pequenos e com mensagens claras.
- Adicione testes quando fizer correções comportamentais significativas.
- Execute `go build ./...` antes de abrir o PR e confirme que o projeto compila.

Sugestões de revisão

- Prefira mudanças que mantenham o comportamento atual por padrão e adicionem
  uma rota/flag para o comportamento corrigido (isso facilita comparações).
- Documente benchmarks e passos para reproduzir em `README.md`.

Licença

Ao enviar um pull request você concorda em licenciar suas contribuições sob a
mesma licença do projeto (MIT).
