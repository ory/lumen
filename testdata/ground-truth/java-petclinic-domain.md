<!-- Source: https://github.com/spring-projects/spring-petclinic -->
<!-- Validated against fixtures: 2026-03-05 -->

## Reference Documentation

Spring PetClinic is a sample Spring Boot application demonstrating JPA/Hibernate
entity mapping, Spring Data repositories, and Spring MVC controllers. The domain
model uses a three-branch inheritance hierarchy rooted at BaseEntity, with
@MappedSuperclass for abstract types and @Entity for concrete persistent types.

Relationships use JPA annotations: @OneToMany for Owner→Pet and Pet→Visit,
@ManyToOne for Pet→PetType, and @ManyToMany for Vet→Specialty. All collections
use EAGER fetching. The repository layer uses Spring Data interfaces
(JpaRepository and Repository) with derived query methods and @Query annotations.

## Key Types in Fixtures

**Entity hierarchy (model package):**
- `BaseEntity` (BaseEntity.java) — @MappedSuperclass, root with @Id Integer id
- `NamedEntity` (NamedEntity.java) — @MappedSuperclass, extends BaseEntity, adds String name
- `Person` (Person.java) — @MappedSuperclass, extends BaseEntity, adds firstName/lastName

**Concrete entities (owner package):**
- `Owner` (Owner.java) — @Entity @Table("owners"), extends Person
- `Pet` (Pet.java) — @Entity @Table("pets"), extends NamedEntity
- `PetType` (PetType.java) — @Entity @Table("types"), extends NamedEntity
- `Visit` (Visit.java) — @Entity @Table("visits"), extends BaseEntity

**Concrete entities (vet package):**
- `Vet` (Vet.java) — @Entity @Table("vets"), extends Person
- `Specialty` (Specialty.java) — @Entity @Table("specialties"), extends NamedEntity

**Repositories:**
- `OwnerRepository` (OwnerRepository.java) — extends JpaRepository<Owner, Integer>
- `PetTypeRepository` (PetTypeRepository.java) — extends JpaRepository<PetType, Integer>
- `VetRepository` (VetRepository.java) — extends Repository<Vet, Integer>

**Controllers:**
- `OwnerController` (OwnerController.java)
- `PetController` (PetController.java)
- `VetController` (VetController.java)
- `VisitController` (VisitController.java)
- `CrashController` (CrashController.java)

**Other:**
- `Vets` (Vets.java) — XML wrapper DTO, not an entity
- `PetClinicApplication` (PetClinicApplication.java) — @SpringBootApplication main class
- `CacheConfiguration` (CacheConfiguration.java) — @Configuration for JCache/Caffeine

## Required Facts

1. BaseEntity is a @MappedSuperclass with an Integer id field using @GeneratedValue(strategy = GenerationType.IDENTITY).
2. The entity hierarchy has two branches: NamedEntity (adds name) and Person (adds firstName, lastName), both extending BaseEntity as @MappedSuperclass.
3. Owner is an @Entity extending Person, mapped to table "owners", with fields: address, city, telephone.
4. Owner has a @OneToMany relationship to Pet with cascade=CascadeType.ALL, fetch=FetchType.EAGER, joined via @JoinColumn(name = "owner_id"), ordered by @OrderBy("name").
5. Pet is an @Entity extending NamedEntity, mapped to table "pets", with a @ManyToOne relationship to PetType via @JoinColumn(name = "type_id").
6. Pet has a @OneToMany relationship to Visit with cascade=CascadeType.ALL, fetch=FetchType.EAGER, joined via @JoinColumn(name = "pet_id"), ordered by @OrderBy("date ASC").
7. Visit is an @Entity extending BaseEntity directly (NOT NamedEntity or Person), with fields: date (LocalDate, column "visit_date") and description (String).
8. Vet is an @Entity extending Person, with a @ManyToMany(fetch=FetchType.EAGER) relationship to Specialty using @JoinTable(name = "vet_specialties").
9. PetType and Specialty both extend NamedEntity and have no additional fields beyond inherited id and name.
10. OwnerRepository extends JpaRepository<Owner, Integer> and declares findByLastNameStartingWith(String, Pageable) using Spring Data naming conventions.
11. VetRepository extends the generic Repository<Vet, Integer> interface (NOT JpaRepository) and annotates findAll() with @Cacheable("vets") and @Transactional(readOnly = true).
12. PetTypeRepository extends JpaRepository<PetType, Integer> and uses an explicit @Query("SELECT ptype FROM PetType ptype ORDER BY ptype.name") for findPetTypes().
13. Owner.telephone has a @Pattern(regexp = "\\d{10}") validation constraint.
14. All @NotBlank validations are used on: name (NamedEntity), firstName/lastName (Person), address/city/telephone (Owner), description (Visit).

## Hallucination Traps

- There is NO `PetRepository` interface in the fixtures.
- There is NO `VisitRepository` interface in the fixtures.
- There is NO `SpecialtyRepository` interface in the fixtures.
- There is NO `SpecialtyController` in the fixtures.
- Visit does NOT have a back-reference to Pet or to Vet (relationships are unidirectional).
- There is NO @GenerationType.AUTO used anywhere (only IDENTITY).
- There is NO @Version (optimistic locking) on any entity.
- There is NO @EnableJpaRepositories annotation in the fixtures.
- Vets.java is a DTO wrapper class with @XmlRootElement, NOT an entity.
